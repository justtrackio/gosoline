package parquet_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Mocks "github.com/justtrackio/gosoline/pkg/cloud/aws/s3/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/parquet"
	"github.com/stretchr/testify/assert"
)

func TestNopRecorder(t *testing.T) {
	r := parquet.NewNopRecorder()

	r.RecordFile("bucket", "file")
	assert.Equal(t, parquet.NewNopRecorder(), r)

	err := r.DeleteRecordedFiles(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, parquet.NewNopRecorder(), r)

	r.RecordFile("bucket", "another file")
	err = r.RenameRecordedFiles(t.Context(), "prefix")
	assert.NoError(t, err)
	assert.Equal(t, parquet.NewNopRecorder(), r)
}

func TestS3FileRecorder(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	s3Client := s3Mocks.NewClient(t)

	r := parquet.NewS3FileRecorderWithInterfaces(logger, s3Client)
	r.RecordFile("bucket", "my file")
	r.RecordFile("another bucket", "my other file")

	s3Client.EXPECT().DeleteObject(t.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String("bucket"),
		Key:    aws.String("my file"),
	}).Return(nil, nil).Once()
	s3Client.EXPECT().DeleteObject(t.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String("another bucket"),
		Key:    aws.String("my other file"),
	}).Return(nil, nil).Once()

	err := r.DeleteRecordedFiles(t.Context())
	assert.NoError(t, err)

	r.RecordFile("new bucket", "last file")

	s3Client.EXPECT().CopyObject(t.Context(), &s3.CopyObjectInput{
		Bucket:     aws.String("new bucket"),
		Key:        aws.String("prefix/last file"),
		CopySource: aws.String("new bucket/last file"),
	}).Return(nil, nil).Once()
	s3Client.EXPECT().DeleteObject(t.Context(), &s3.DeleteObjectInput{
		Bucket: aws.String("new bucket"),
		Key:    aws.String("last file"),
	}).Return(nil, nil).Once()

	err = r.RenameRecordedFiles(t.Context(), "prefix")
	assert.NoError(t, err)
}
