package parquet

import (
	"context"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/reader"
	"golang.org/x/sync/semaphore"
	"reflect"
	"sync"
	"time"
)

type Progress struct {
	FileCount int
	Current   int
}

type ResultCallback func(progress Progress, result interface{}) (bool, error)

type Reader interface {
	ReadDate(ctx context.Context, datetime time.Time, result interface{}) error
	ReadDateAsync(ctx context.Context, datetime time.Time, result interface{}, callback ResultCallback) error
	ReadFile(ctx context.Context, file string, result interface{}) error
}

type s3Reader struct {
	logger   mon.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	settings *Settings
}

func NewReader(config cfg.Config, logger mon.Logger, settings *Settings) *s3Reader {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)

	return NewReaderWithInterfaces(logger, s3Cfg, s3Client, settings)
}

func NewReaderWithInterfaces(logger mon.Logger, s3Cfg *aws.Config, s3Client s3iface.S3API, settings *Settings) *s3Reader {
	return &s3Reader{
		logger:   logger,
		s3Cfg:    s3Cfg,
		s3Client: s3Client,
		settings: settings,
	}
}

func (r *s3Reader) ReadDate(ctx context.Context, datetime time.Time, result interface{}) error {
	prefix := s3PrefixNamingStrategy(r.settings.ModelId, datetime)
	files, err := r.listFiles(prefix)

	if err != nil {
		return err
	}

	tmp := make(map[int]interface{})
	err = r.ReadDateAsync(ctx, datetime, result, func(progress Progress, result interface{}) (bool, error) {
		tmp[progress.Current] = result
		return true, nil
	})

	if err != nil {
		return err
	}

	pr := reflect.ValueOf(result).Elem()
	for i := range files {
		pt := reflect.ValueOf(tmp[i]).Elem()

		for i := 0; i < pt.Len(); i++ {
			pr.Set(reflect.Append(pr, pt.Index(i)))
		}
	}

	return nil
}

func (r *s3Reader) ReadDateAsync(ctx context.Context, datetime time.Time, result interface{}, callback ResultCallback) error {
	prefix := s3PrefixNamingStrategy(r.settings.ModelId, datetime)
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

			ptr := createPointerToSliceOfTypeAndSize(result, 0)
			err := r.ReadFile(ctx, file, ptr)

			if err != nil {
				r.logger.Fatal(err, "can not read file")
				return
			}

			ok, err := callback(Progress{
				FileCount: fileCount,
				Current:   i,
			}, ptr)

			if !ok {
				stop = true
				return
			}
		}(i, file)
	}

	wg.Wait()

	return nil
}

func (r *s3Reader) ReadFile(ctx context.Context, file string, result interface{}) error {
	bucket := r.getBucketName()
	fr, err := parquetS3.NewS3FileReader(ctx, bucket, file, r.s3Cfg)

	if err != nil {
		return err
	}

	schemaTyp := findBaseType(result)
	schema := reflect.New(schemaTyp).Interface()

	pr, err := reader.NewParquetReader(fr, schema, 4)

	if err != nil {
		return err
	}

	size := int(pr.GetNumRows())
	ptr := createPointerToSliceOfTypeAndSize(result, size)

	if err = pr.Read(ptr); err != nil {
		return err
	}

	pr.ReadStop()
	if err = fr.Close(); err != nil {
		return err
	}

	copyPointerSlice(result, ptr)

	return nil
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

func (r *s3Reader) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     r.settings.ModelId.Project,
		Environment: r.settings.ModelId.Environment,
		Family:      r.settings.ModelId.Family,
		Application: r.settings.ModelId.Application,
	})
}
