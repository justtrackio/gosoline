package parquet

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name FileRecorder
type FileRecorder interface {
	DeleteRecordedFiles(ctx context.Context) error
	Files() []File
	RecordFile(bucket string, key string)
	RenameRecordedFiles(ctx context.Context, newPrefix string) error
}

type File struct {
	Bucket string
	Key    string
}

type s3FileRecorder struct {
	logger   log.Logger
	files    []File
	lck      sync.Mutex
	s3Client gosoS3.Client
}

type nopRecorder struct{}

func (w nopRecorder) Files() []File {
	return nil
}

func (w nopRecorder) RecordFile(_ string, _ string) {
}

func (w nopRecorder) RenameRecordedFiles(_ context.Context, _ string) error {
	return nil
}

func (w nopRecorder) DeleteRecordedFiles(_ context.Context) error {
	return nil
}

func NewNopRecorder() FileRecorder {
	return nopRecorder{}
}

func NewS3FileRecorder(ctx context.Context, config cfg.Config, logger log.Logger, name string) (FileRecorder, error) {
	if name == "" {
		name = "default"
	}

	s3Client, err := gosoS3.ProvideClient(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create s3 client with name %s: %w", name, err)
	}

	return NewS3FileRecorderWithInterfaces(logger, s3Client), nil
}

func NewS3FileRecorderWithInterfaces(logger log.Logger, s3Client gosoS3.Client) FileRecorder {
	return &s3FileRecorder{
		logger:   logger,
		s3Client: s3Client,
		lck:      sync.Mutex{},
		files:    make([]File, 0),
	}
}

func (w *s3FileRecorder) Files() []File {
	return w.files
}

func (w *s3FileRecorder) RecordFile(bucket string, key string) {
	w.lck.Lock()
	defer w.lck.Unlock()

	w.files = append(w.files, File{
		Bucket: bucket,
		Key:    key,
	})
}

func (w *s3FileRecorder) RenameRecordedFiles(ctx context.Context, newPrefix string) error {
	w.lck.Lock()
	defer w.lck.Unlock()

	for _, file := range w.files {
		copyObjectInput := &s3.CopyObjectInput{
			Bucket:     aws.String(file.Bucket),
			Key:        aws.String(fmt.Sprintf("%s/%s", newPrefix, file.Key)),
			CopySource: aws.String(fmt.Sprintf("%s/%s", file.Bucket, file.Key)),
		}

		if _, err := w.s3Client.CopyObject(ctx, copyObjectInput); err != nil {
			return err
		}

		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(file.Bucket),
			Key:    aws.String(file.Key),
		}

		if _, err := w.s3Client.DeleteObject(ctx, deleteObjectInput); err != nil {
			return err
		}
	}

	w.files = make([]File, 0)

	return nil
}

func (w *s3FileRecorder) DeleteRecordedFiles(ctx context.Context) error {
	w.lck.Lock()
	defer w.lck.Unlock()

	for _, file := range w.files {
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: &file.Bucket,
			Key:    &file.Key,
		}

		if _, err := w.s3Client.DeleteObject(ctx, deleteObjectInput); err != nil {
			return err
		}
	}

	w.files = make([]File, 0)

	return nil
}
