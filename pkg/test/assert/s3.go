package assert

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	"testing"
)

func S3BucketExists(t *testing.T, client *s3.S3, bucketName string) {
	getTopicAttributesOutput, err := client.ListBuckets(&s3.ListBucketsInput{})

	assert.NotNil(t, getTopicAttributesOutput)
	assert.NoError(t, err)
	b := funk.Find(getTopicAttributesOutput.Buckets, func(bucket *s3.Bucket) bool {
		return bucket.Name != nil && *bucket.Name == bucketName
	})

	assert.IsType(t, &s3.Bucket{}, b)
}
