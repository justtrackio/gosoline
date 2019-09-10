package parquet

import (
	"context"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/writer"
	"reflect"
	"time"
)

type WriterSettings struct {
	ModelId  mdl.ModelId
	Interval time.Duration
}

type Writer interface {
	Write(ctx context.Context, items interface{}) error
}

type s3Writer struct {
	logger mon.Logger
	s3Cfg  *aws.Config

	settings *WriterSettings
}

func NewWriter(config cfg.Config, logger mon.Logger, settings *WriterSettings) *s3Writer {
	s3Cfg := blob.GetS3ClientConfig(config)
	settings.ModelId.PadFromConfig(config)

	return NewWriterWithInterfaces(logger, s3Cfg, settings)
}

func NewWriterWithInterfaces(logger mon.Logger, s3Cfg *aws.Config, settings *WriterSettings) *s3Writer {
	return &s3Writer{
		logger:   logger,
		s3Cfg:    s3Cfg,
		settings: settings,
	}
}

func (w *s3Writer) Write(ctx context.Context, items interface{}) error {
	current := time.Time{}
	buckets := make(map[time.Time][]TimeStampable)

	val := reflect.ValueOf(items)

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface().(TimeStampable)

		if current.IsZero() || item.GetCreatedAt().Sub(current) > w.settings.Interval {
			current = item.GetCreatedAt()
		}

		if _, ok := buckets[current]; !ok {
			buckets[current] = make([]TimeStampable, 0)
		}

		buckets[current] = append(buckets[current], item)
	}

	for i, bucket := range buckets {
		err := w.writeBucket(ctx, i, items, bucket)

		if err != nil {
			return err
		}
	}

	return nil
}

func (w *s3Writer) writeBucket(ctx context.Context, datetime time.Time, rootItems interface{}, items []TimeStampable) error {
	bucket := w.getBucketName()
	key := s3KeyNamingStrategy(w.settings.ModelId, datetime)

	fw, err := parquetS3.NewS3FileWriter(ctx, bucket, key, []func(*s3manager.Uploader){})

	if err != nil {
		return err
	}

	schemaTyp := findBaseType(rootItems)
	schema := reflect.New(schemaTyp).Interface()

	pw, err := writer.NewParquetWriter(fw, schema, 4)

	if err != nil {
		return err
	}

	for _, item := range items {
		if err = pw.Write(item); err != nil {
			return err
		}
	}

	if err = pw.WriteStop(); err != nil {
		return err
	}

	if err = fw.Close(); err != nil {
		return err
	}

	return nil
}

func (w *s3Writer) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     w.settings.ModelId.Project,
		Environment: w.settings.ModelId.Environment,
		Family:      w.settings.ModelId.Family,
		Application: w.settings.ModelId.Application,
	})
}
