package parquet

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"sync"
)

//go:generate mockery -name FileRecorder
type FileRecorder interface {
	RecordFile(bucket string, key string)
	RenameRecordedFiles(ctx context.Context, newPrefix string) error
	DeleteRecordedFiles(ctx context.Context) error
}

type File struct {
	Bucket string
	Key    string
}

type s3FileRecorder struct {
	logger   log.Logger
	s3Client s3iface.S3API
	lck      sync.Mutex
	files    []File
}

type nopRecorder struct {
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

func NewS3FileRecorder(config cfg.Config, logger log.Logger) FileRecorder {
	s3Client := blob.ProvideS3Client(config)

	return NewS3FileRecorderWithInterfaces(logger, s3Client)
}

func NewS3FileRecorderWithInterfaces(logger log.Logger, s3Client s3iface.S3API) FileRecorder {
	return &s3FileRecorder{
		logger:   logger,
		s3Client: s3Client,
		lck:      sync.Mutex{},
		files:    make([]File, 0),
	}
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

		if _, err := w.s3Client.CopyObjectWithContext(ctx, copyObjectInput); err != nil {
			return err
		}

		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(file.Bucket),
			Key:    aws.String(file.Key),
		}

		if _, err := w.s3Client.DeleteObjectWithContext(ctx, deleteObjectInput); err != nil {
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

		if _, err := w.s3Client.DeleteObjectWithContext(ctx, deleteObjectInput); err != nil {
			return err
		}
	}

	w.files = make([]File, 0)

	return nil
}
