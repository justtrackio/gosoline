package parquet

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/reader"
	"golang.org/x/sync/semaphore"
	"reflect"
	"strings"
	"time"
)

type Progress struct {
	FileCount int
	Current   int
}

type ReadResult map[string]interface{}
type ReadResults []ReadResult
type ResultCallback func(progress Progress, results interface{}) (bool, error)

//go:generate mockery -name Reader
type Reader interface {
	ReadDate(ctx context.Context, datetime time.Time, target interface{}) error
	ReadDateAsync(ctx context.Context, datetime time.Time, target interface{}, callback ResultCallback) error
	ReadFile(ctx context.Context, file string) (ReadResults, error)
}

type s3Reader struct {
	logger   log.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	modelId              mdl.ModelId
	prefixNamingStrategy S3PrefixNamingStrategy
	recorder             FileRecorder
}

func NewReader(config cfg.Config, logger log.Logger, settings *ReaderSettings) *s3Reader {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		panic(fmt.Sprintf("Unknown prefix naming strategy '%s'", settings.NamingStrategy))
	}

	recorder := settings.Recorder
	if recorder == nil {
		recorder = NewNopRecorder()
	}

	return NewReaderWithInterfaces(logger, s3Cfg, s3Client, settings.ModelId, prefixNaming, recorder)
}

func NewReaderWithInterfaces(
	logger log.Logger,
	s3Cfg *aws.Config,
	s3Client s3iface.S3API,
	modelId mdl.ModelId,
	prefixNaming S3PrefixNamingStrategy,
	recorder FileRecorder,
) *s3Reader {
	return &s3Reader{
		logger:               logger,
		s3Cfg:                s3Cfg,
		s3Client:             s3Client,
		modelId:              modelId,
		prefixNamingStrategy: prefixNaming,
		recorder:             recorder,
	}
}

func (r *s3Reader) ReadDate(ctx context.Context, datetime time.Time, target interface{}) error {
	if !refl.IsPointerToSlice(target) {
		return fmt.Errorf("target needs to be a pointer to a slice, but is %T", target)
	}

	tmp := refl.CreatePointerToSliceOfTypeAndSize(target, 0)
	tp := reflect.ValueOf(tmp)
	t := tp.Elem()

	err := r.ReadDateAsync(ctx, datetime, target, func(progress Progress, result interface{}) (bool, error) {
		rp := reflect.ValueOf(result)
		r := rp.Elem()

		t.Set(reflect.AppendSlice(t, r))

		return true, nil
	})

	if err != nil {
		return err
	}

	refl.CopyPointerSlice(target, tmp)

	return nil
}

func (r *s3Reader) ReadDateAsync(ctx context.Context, datetime time.Time, target interface{}, callback ResultCallback) error {
	if !refl.IsPointerToSlice(target) {
		return fmt.Errorf("target needs to be a pointer to a slice, but is %T", target)
	}

	files, err := r.listFilesFromDate(datetime)
	if err != nil {
		return err
	}

	fileCount := len(files)
	if fileCount == 0 {
		return nil
	}

	stop := false

	sem := semaphore.NewWeighted(int64(10))
	cfn := coffin.New()

	for i, file := range files {
		err := sem.Acquire(ctx, int64(1))

		if err != nil {
			return err
		}

		cfn.GoWithContextf(ctx, func(i int, file string) func(ctx context.Context) error {
			return func(ctx context.Context) error {
				defer sem.Release(int64(1))

				if stop {
					return nil
				}

				r.logger.Debug("reading file %d of %d: %s", i, fileCount, file)
				result, err := r.ReadFile(ctx, file)

				if err != nil {
					return fmt.Errorf("can not read file %s: %w", file, err)
				}

				decoded := refl.CreatePointerToSliceOfTypeAndSize(target, len(result))
				err = r.decode(result, decoded)

				if err != nil {
					return fmt.Errorf("can not decode results in file %s: %w", file, err)
				}

				ok, err := callback(Progress{FileCount: fileCount, Current: i}, decoded)

				if err != nil {
					return fmt.Errorf("callback failed: %w", err)
				}

				if !ok {
					stop = true
					return nil
				}

				r.recorder.RecordFile(r.getBucketName(), file)

				return nil
			}
		}(i, file), "panic during file read")
	}

	return cfn.Wait()
}

func (r *s3Reader) ReadFile(ctx context.Context, file string) (ReadResults, error) {
	bucket := r.getBucketName()
	fr, err := parquetS3.NewS3FileReader(ctx, bucket, file, r.s3Cfg)

	if err != nil {
		return nil, err
	}

	pr, err := reader.NewParquetColumnReader(fr, 4)

	if err != nil {
		return nil, err
	}

	size := int(pr.GetNumRows())
	columnNames := pr.SchemaHandler.ValueColumns
	columns := make(map[string][]interface{})
	rootName := pr.Footer.Schema[0].Name

	for _, colSchema := range columnNames {
		values, _, _, err := pr.ReadColumnByPath(colSchema, size)

		if err != nil {
			return nil, err
		}

		key := strings.ToLower(colSchema[len(rootName)+1:])

		columns[key] = values
	}

	pr.ReadStop()
	if err = fr.Close(); err != nil {
		return nil, err
	}

	results := make(ReadResults, size)

	for key, values := range columns {
		for i := 0; i < size; i++ {
			if results[i] == nil {
				results[i] = make(ReadResult, len(columns))
			}

			results[i][key] = values[i]
		}
	}

	return results, nil
}

func (r *s3Reader) listFilesFromDate(datetime time.Time) ([]string, error) {
	prefix := r.prefixNamingStrategy(r.modelId, datetime)
	files, err := r.listFiles(prefix)

	if err != nil {
		return nil, err
	}

	return files, err
}

func (r *s3Reader) listFiles(prefix string) ([]string, error) {
	bucket := r.getBucketName()

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	files := make([]string, 0, 128)

	for {
		out, err := r.s3Client.ListObjects(input)

		if err != nil {
			return nil, err
		}

		if len(out.Contents) == 0 {
			break
		}

		for _, obj := range out.Contents {
			files = append(files, *obj.Key)
		}

		if !*out.IsTruncated {
			break
		}

		input.Marker = out.Contents[len(out.Contents)-1].Key
	}

	return files, nil
}

func (r *s3Reader) decode(input interface{}, output interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          "parquet",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			r.decodeTimeMillisHook(), // used to decode firehose parquet time millis (little endian int96)
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)

	if err != nil {
		return errors.Wrap(err, "can not initialize decoder")
	}

	err = decoder.Decode(input)

	if err != nil {
		return errors.Wrap(err, "can not decode input")
	}

	return nil
}

func (r *s3Reader) decodeTimeMillisHook() interface{} {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		return parseInt96Timestamp(data.(string)), nil
	}
}

func (r *s3Reader) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     r.modelId.Project,
		Environment: r.modelId.Environment,
		Family:      r.modelId.Family,
		Application: r.modelId.Application,
	})
}
