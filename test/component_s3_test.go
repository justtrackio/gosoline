//go:build integration
// +build integration

package test_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	gosoAssert "github.com/justtrackio/gosoline/pkg/test/assert"
	"github.com/stretchr/testify/assert"
)

func Test_s3(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.s3.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	s3Client := mocks.ProvideS3Client("s3")
	o, err := s3Client.ListBuckets(&s3.ListBucketsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.Buckets, 0)

	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String("foo"),
	})

	assert.NoError(t, err)
	gosoAssert.S3BucketExists(t, s3Client, "foo")
}
