package parquet

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	refl "github.com/applike/gosoline/pkg/refl"
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
	"sync"
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
	logger   mon.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	prefixNamingStrategy s3PrefixNamingStrategy

	settings *ReaderSettings
}

func NewReader(config cfg.Config, logger mon.Logger, settings *ReaderSettings) *s3Reader {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		panic(fmt.Sprintf("Unknown prefix naming strategy '%s'", settings.NamingStrategy))
	}

	return NewReaderWithInterfaces(logger, s3Cfg, s3Client, prefixNaming, settings)
}

func NewReaderWithInterfaces(logger mon.Logger, s3Cfg *aws.Config, s3Client s3iface.S3API, prefixNaming s3PrefixNamingStrategy, settings *ReaderSettings) *s3Reader {
	return &s3Reader{
		logger:               logger,
		s3Cfg:                s3Cfg,
		s3Client:             s3Client,
		prefixNamingStrategy: prefixNaming,
		settings:             settings,
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

	prefix := r.prefixNamingStrategy(r.settings.ModelId, datetime)
	files, err := r.listFiles(prefix)

	if err != nil {
		return err
	}

	fileCount := len(files)
	stop := false

	sem := semaphore.NewWeighted(int64(10))
	wg := sync.WaitGroup{}
	wg.Add(fileCount)

	for i, file := range files {
		err := sem.Acquire(ctx, int64(1))

		if err != nil {
			return err
		}

		go func(i int, file string) {
			defer sem.Release(int64(1))
			defer wg.Done()

			if stop {
				return
			}

			r.logger.Debugf("reading file %d of %d: %s", i, fileCount, file)

			result, err := r.ReadFile(ctx, file)

			if err != nil {
				r.logger.Fatal(err, "can not read file")
				return
			}

			decoded := refl.CreatePointerToSliceOfTypeAndSize(target, len(result))
			err = r.decode(result, decoded)

			if err != nil {
				r.logger.Error(err, "could not decode results")
				return
			}

			ok, err := callback(Progress{
				FileCount: fileCount,
				Current:   i,
			}, decoded)

			if !ok {
				stop = true
				return
			}
		}(i, file)
	}

	wg.Wait()

	return nil
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
			r.decodeTimeMillisHook(),      // used to decode firehose timestamps
			r.decodeTimeMillisHookInt64(), // used to decode manually written timestamps
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

func (r *s3Reader) decodeTimeMillisHookInt64() interface{} {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.Int64 || t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		// when writing time stamps via parquet.writer the parquet package recommended writing it as int64
		// unless we figure out how to write the firehose format,
		// we need this to decode those manually written timestamps

		parquetDate := data.(int64)
		seconds := parquetDate / 1000
		tm := time.Unix(seconds, 0)

		return tm, nil
	}
}

func (r *s3Reader) decodeTimeMillisHook() interface{} {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		// firehose stores timestamp_millis as nano + julian date int96 values
		// the corresponding go type would be a 12 byte byte array, with
		// - the first 8 bytes are the nanoseconds within the day
		// - the following four bytes represent the julian date

		parquetDate := []byte(data.(string))

		nano := binary.LittleEndian.Uint64(parquetDate[:8])
		dt := binary.LittleEndian.Uint32(parquetDate[8:])

		// julian date to Y-m-d conversion based on https://github.com/carlosjhr64/jd/blob/master/jd.go#L24

		l := dt + 68569
		n := 4 * l / 146097
		l = l - (146097*n+3)/4
		i := 4000 * (l + 1) / 1461001
		l = l - 1461*i/4 + 31
		j := 80 * l / 2447
		k := l - 2447*j/80
		l = j / 11
		j = j + 2 - 12*l
		i = 100*(n-49) + i + l

		tm := time.Date(int(i), time.Month(j), int(k), 0, 0, 0, 0, time.UTC)
		tm = tm.Add(time.Duration(nano))

		return tm, nil
	}
}

func (r *s3Reader) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     r.settings.ModelId.Project,
		Environment: r.settings.ModelId.Environment,
		Family:      r.settings.ModelId.Family,
		Application: r.settings.ModelId.Application,
	})
}
