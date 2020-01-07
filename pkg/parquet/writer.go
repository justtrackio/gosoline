package parquet

import (
	"context"
	"encoding/json"
	"fmt"
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
	ModelId        mdl.ModelId
	Interval       time.Duration
	NamingStrategy string
}

type Writer interface {
	Write(ctx context.Context, items interface{}) error
}

type s3Writer struct {
	logger mon.Logger
	s3Cfg  *aws.Config

	prefixNamingStrategy s3PrefixNamingStrategy

	settings *WriterSettings
}

func NewWriter(config cfg.Config, logger mon.Logger, settings *WriterSettings) *s3Writer {
	s3Cfg := blob.GetS3ClientConfig(config)
	settings.ModelId.PadFromConfig(config)

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		panic(fmt.Sprintf("Unknown prefix naming strategy '%s'", settings.NamingStrategy))
	}

	return NewWriterWithInterfaces(logger, s3Cfg, prefixNaming, settings)
}

func NewWriterWithInterfaces(logger mon.Logger, s3Cfg *aws.Config, prefixNaming s3PrefixNamingStrategy, settings *WriterSettings) *s3Writer {
	return &s3Writer{
		logger:               logger,
		s3Cfg:                s3Cfg,
		prefixNamingStrategy: prefixNaming,
		settings:             settings,
	}
}

func (w *s3Writer) Write(ctx context.Context, datetime time.Time, items interface{}) error {
	bucket := w.getBucketName()
	key := s3KeyNamingStrategy(w.settings.ModelId, datetime, w.prefixNamingStrategy)

	schema, converted, err := w.parseItems(items)

	if err != nil {
		return err
	}

	fw, err := parquetS3.NewS3FileWriter(ctx, bucket, key, []func(*s3manager.Uploader){}, w.s3Cfg)

	if err != nil {
		return err
	}

	pw, err := writer.NewJSONWriter(schema, fw, 4)

	if err != nil {
		return err
	}

	for _, item := range converted {
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

func (w *s3Writer) parseItems(items interface{}) (string, []string, error) {
	schema, err := parseSchema(items)

	if err != nil {
		return "", nil, fmt.Errorf("could not parse schema: %w", err)
	}

	it := reflect.ValueOf(items).Elem()

	converted := make([]string, it.Len())

	for i := 0; i < it.Len(); i++ {
		item := it.Index(i).Interface()

		m, err := mapFieldsToTags(item)

		if err != nil {
			return "", nil, fmt.Errorf("could not map fields to tags: %w", err)
		}

		marshalled, err := json.Marshal(m)

		if err != nil {
			return "", nil, fmt.Errorf("could not marshal mapped item: %w", err)
		}

		converted[i] = string(marshalled)
	}

	return schema, converted, nil
}

func (w *s3Writer) getBucketName() string {
	return s3BucketNamingStrategy(cfg.AppId{
		Project:     w.settings.ModelId.Project,
		Environment: w.settings.ModelId.Environment,
		Family:      w.settings.ModelId.Family,
		Application: w.settings.ModelId.Application,
	})
}
