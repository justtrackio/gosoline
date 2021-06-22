package parquet_test

import (
	"context"
	"github.com/applike/gosoline/pkg/blob/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/parquet"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNopRecorder(t *testing.T) {
	r := parquet.NewNopRecorder()

	r.RecordFile("bucket", "file")
	assert.Equal(t, parquet.NewNopRecorder(), r)

	err := r.DeleteRecordedFiles(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, parquet.NewNopRecorder(), r)

	r.RecordFile("bucket", "another file")
	err = r.RenameRecordedFiles(context.TODO(), "prefix")
	assert.NoError(t, err)
	assert.Equal(t, parquet.NewNopRecorder(), r)
}

func TestS3FileRecorder(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	s3Api := new(mocks.S3API)

	r := parquet.NewS3FileRecorderWithInterfaces(logger, s3Api)
	r.RecordFile("bucket", "my file")
	r.RecordFile("another bucket", "my other file")

	s3Api.On("DeleteObjectWithContext", context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String("bucket"),
		Key:    aws.String("my file"),
	}).Return(nil, nil).Once()
	s3Api.On("DeleteObjectWithContext", context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String("another bucket"),
		Key:    aws.String("my other file"),
	}).Return(nil, nil).Once()

	err := r.DeleteRecordedFiles(context.TODO())
	assert.NoError(t, err)
	s3Api.AssertExpectations(t)

	r.RecordFile("new bucket", "last file")

	s3Api.On("CopyObjectWithContext", context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String("new bucket"),
		Key:        aws.String("prefix/last file"),
		CopySource: aws.String("new bucket/last file"),
	}).Return(nil, nil).Once()
	s3Api.On("DeleteObjectWithContext", context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String("new bucket"),
		Key:    aws.String("last file"),
	}).Return(nil, nil).Once()

	err = r.RenameRecordedFiles(context.TODO(), "prefix")
	assert.NoError(t, err)
	s3Api.AssertExpectations(t)
}
