package parquet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	parquetS3 "github.com/xitongsys/parquet-go-source/s3"
	"github.com/xitongsys/parquet-go/writer"
	"time"
)

type WriterSettings struct {
	ModelId        mdl.ModelId
	NamingStrategy string
	Tags           map[string]string
}

//go:generate mockery -name Writer
type Writer interface {
	Write(ctx context.Context, datetime time.Time, items interface{}) error
	DeleteReadFiles(ctx context.Context) error
	DeleteWrittenFiles(ctx context.Context) error
}

type s3Writer struct {
	logger   mon.Logger
	s3Cfg    *aws.Config
	s3Client s3iface.S3API

	prefixNamingStrategy s3PrefixNamingStrategy

	settings     *WriterSettings
	tags         map[string]string
	writtenFiles []string
}

func NewWriter(config cfg.Config, logger mon.Logger, settings *WriterSettings) *s3Writer {
	s3Cfg := blob.GetS3ClientConfig(config)
	s3Client := blob.ProvideS3Client(config)
	settings.ModelId.PadFromConfig(config)

	prefixNaming, exists := s3PrefixNamingStrategies[settings.NamingStrategy]

	if !exists {
		logger.Panic(errors.New("unknown naming strategy"), fmt.Sprintf("Unknown prefix naming strategy '%s'", settings.NamingStrategy))
	}

	var writtenFileKeys []string

	return NewWriterWithInterfaces(logger, s3Client, s3Cfg, prefixNaming, settings, writtenFileKeys)
}

func NewWriterWithInterfaces(logger mon.Logger, s3Client s3iface.S3API, s3Cfg *aws.Config, prefixNaming s3PrefixNamingStrategy, settings *WriterSettings, writtenFileKeys []string) *s3Writer {
	tags := map[string]string{
		"Project":     settings.ModelId.Project,
		"Environment": settings.ModelId.Environment,
		"Family":      settings.ModelId.Family,
		"Application": settings.ModelId.Application,
		"Model":       settings.ModelId.Name,
	}

	for k, v := range settings.Tags {
		tags[k] = v
	}

	return &s3Writer{
		logger:               logger,
		s3Cfg:                s3Cfg,
		s3Client:             s3Client,
		prefixNamingStrategy: prefixNaming,
		settings:             settings,
		tags:                 tags,
		writtenFiles:         writtenFileKeys,
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

	w.writtenFiles = append(w.writtenFiles, key)

	tagSet := makeTags(w.tags)

	if len(tagSet) == 0 {
		return nil
	}

	tagInput := &s3.PutObjectTaggingInput{
		Bucket:  &bucket,
		Key:     &key,
		Tagging: &s3.Tagging{TagSet: tagSet},
	}

	if _, err := w.s3Client.PutObjectTaggingWithContext(ctx, tagInput); err != nil {
		return err
	}

	return nil
}

func (w *s3Writer) parseItems(items interface{}) (string, []string, error) {
	schema, err := parseSchema(items)

	if err != nil {
		return "", nil, fmt.Errorf("could not parse schema: %w", err)
	}

	it := refl.SliceInterfaceIterator(items)
	converted := make([]string, 0, it.Len())

	for it.Next() {
		item := it.Val()

		m, err := mapFieldsToTags(item)

		if err != nil {
			return "", nil, fmt.Errorf("could not map fields to tags: %w", err)
		}

		marshalled, err := json.Marshal(m)

		if err != nil {
			return "", nil, fmt.Errorf("could not marshal mapped item: %w", err)
		}

		converted = append(converted, string(marshalled))
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

func makeTags(tags map[string]string) []*s3.Tag {
	s3Tags := make([]*s3.Tag, 0, len(tags))

	for key, value := range tags {
		s3Tags = append(s3Tags, &s3.Tag{
			Key:   mdl.String(key),
			Value: mdl.String(value),
		})
	}

	return s3Tags
}

func (w *s3Writer) DeleteReadFiles(ctx context.Context) error {
	bucket := w.getBucketName()

	for _, file := range ReadS3Files {
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &file,
		}

		if _, err := w.s3Client.DeleteObjectWithContext(ctx, deleteObjectInput); err != nil {
			return err
		}
	}

	return nil
}

func (w *s3Writer) DeleteWrittenFiles(ctx context.Context) error {
	bucket := w.getBucketName()

	for _, file := range w.writtenFiles {
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: &bucket,
			Key:    &file,
		}

		if _, err := w.s3Client.DeleteObjectWithContext(ctx, deleteObjectInput); err != nil {
			return err
		}
	}

	return nil
}
