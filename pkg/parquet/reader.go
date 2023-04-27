package parquet

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3v2"
	"github.com/xitongsys/parquet-go/common"
	"github.com/xitongsys/parquet-go/reader"
	"golang.org/x/sync/semaphore"
)

type Progress struct {
	Current   int
	FileCount int
}

type (
	ReadResult     map[string]interface{}
	ReadResults    []ReadResult
	ResultCallback func(progress Progress, results interface{}) (bool, error)
)

//go:generate mockery --name Reader
type Reader interface {
	ReadDate(ctx context.Context, datetime time.Time, target interface{}) error
	ReadDateAsync(ctx context.Context, datetime time.Time, target interface{}, callback ResultCallback) error
	ReadFileIntoTarget(ctx context.Context, file string, target interface{}, batchSize int, offset int64) error
}

type s3Reader struct {
	logger log.Logger

	modelId              mdl.ModelId
	prefixNamingStrategy S3PrefixNamingStrategy
	recorder             FileRecorder
	s3Client             gosoS3.Client
}

func NewReader(ctx context.Context, config cfg.Config, logger log.Logger, settings *ReaderSettings) (Reader, error) {
	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, settings.ClientName)
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client with name %s: %w", settings.ClientName, err)
	}

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		return nil, fmt.Errorf("unknown prefix naming strategy: %s", settings.NamingStrategy)
	}

	recorder := settings.Recorder
	if recorder == nil {
		recorder = NewNopRecorder()
	}

	return NewReaderWithInterfaces(logger, s3Client, settings.ModelId, prefixNaming, recorder), nil
}

func NewReaderWithInterfaces(
	logger log.Logger,
	s3Client gosoS3.Client,
	modelId mdl.ModelId,
	prefixNaming S3PrefixNamingStrategy,
	recorder FileRecorder,
) Reader {
	return &s3Reader{
		logger:               logger,
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

	files, err := r.listFilesFromDate(ctx, datetime)
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

				decoded := refl.CreatePointerToSliceOfTypeAndSize(target, 0)

				err := r.ReadFileIntoTarget(ctx, file, decoded, -1, 0)
				if err != nil {
					return fmt.Errorf("can not read file %s: %w", file, err)
				}

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

func (r *s3Reader) ReadFileIntoTarget(ctx context.Context, file string, target interface{}, batchSize int, offset int64) error {
	if !refl.IsPointerToSlice(target) {
		return fmt.Errorf("target needs to be a pointer to a slice, but is %T", target)
	}

	var columnNames []string
	// as the user should provide *[]T here we have to unwrap the pointer and the slice to get the type of T
	v := reflect.ValueOf(target).Type().Elem().Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// Get the field tag value
		tag := field.Tag.Get("parquet")
		if tag == "" {
			continue
		}

		columnNames = append(columnNames, tag)
	}

	result, err := r.ReadFileColumns(ctx, columnNames, file, batchSize, offset)
	if err != nil {
		return fmt.Errorf("can not read file %s: %w", file, err)
	}

	decoded := refl.CreatePointerToSliceOfTypeAndSize(target, len(result))

	err = r.decode(result, decoded)
	if err != nil {
		return fmt.Errorf("can not decode results in file %s: %w", file, err)
	}

	refl.CopyPointerSlice(target, decoded)

	return nil
}

func (r *s3Reader) ReadFileColumns(ctx context.Context, columnNames []string, file string, batchSize int, offset int64) (ReadResults, error) {
	bucket := r.getBucketName()

	fr, err := parquetS3.NewS3FileReaderWithClient(ctx, r.s3Client, bucket, file)
	if err != nil {
		return nil, err
	}

	pr, err := reader.NewParquetColumnReader(fr, 4)
	if err != nil {
		return nil, err
	}

	size := pr.GetNumRows()
	results := make(ReadResults, size)

	if size == 0 {
		return results, nil
	}

	if offset > 0 {
		err = pr.SkipRows(offset)
		if err != nil {
			return nil, err
		}
	}

	if batchSize > 0 {
		remainingRows := size - offset
		size = int64(batchSize)

		if int64(batchSize) > remainingRows {
			size = remainingRows
		}
	}

	columns := make(map[string][]interface{})

	columnsToRead := funk.SliceToSet(columnNames)

	for column := range columnsToRead {
		path := common.ReformPathStr(fmt.Sprintf("%s.%s", parquetRoot, column))

		values, _, _, err := pr.ReadColumnByPath(path, size)
		if err != nil {
			return nil, err
		}

		columns[column] = values
	}

	pr.ReadStop()
	if err = fr.Close(); err != nil {
		return nil, err
	}

	for key, values := range columns {
		for i := int64(0); i < size; i++ {
			if results[i] == nil {
				results[i] = make(ReadResult, len(columns))
			}

			results[i][key] = values[i]
		}
	}

	return results, nil
}

func (r *s3Reader) listFilesFromDate(ctx context.Context, datetime time.Time) ([]string, error) {
	prefix := r.prefixNamingStrategy(r.modelId, datetime)

	files, err := r.listFiles(ctx, prefix)
	if err != nil {
		return nil, err
	}

	return files, err
}

func (r *s3Reader) listFiles(ctx context.Context, prefix string) ([]string, error) {
	bucket := r.getBucketName()

	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	files := make([]string, 0, 128)

	for {
		out, err := r.s3Client.ListObjects(ctx, input)
		if err != nil {
			return nil, err
		}

		if len(out.Contents) == 0 {
			break
		}

		for _, obj := range out.Contents {
			files = append(files, *obj.Key)
		}

		if !out.IsTruncated {
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
