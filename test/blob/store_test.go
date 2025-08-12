//go:build integration

package blob_test

import (
	"fmt"
	"testing"

	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type StoreTestSuite struct {
	suite.Suite
}

func (s *StoreTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("store_test_cfg.yml"),
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(clock.NewRealClock()),
		suite.WithModule("blob-store-runner-default", blob.ProvideBatchRunner("default")),
	}
}

func (s *StoreTestSuite) TestNewDefault(_ suite.AppUnderTest) {
	client, err := s3.ProvideClient(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default")
	if !s.NoError(err) {
		return
	}

	_, err = client.CreateBucket(s.T().Context(), &awsS3.CreateBucketInput{
		Bucket: mdl.Box("prj-test-fam"),
	})
	if !s.NoError(err) {
		return
	}

	store, err := blob.NewStore(s.Env().Context(), s.Env().Config(), s.Env().Logger(), "default")
	if !s.NoError(err) {
		return
	}

	size := 1001

	batch := make(blob.Batch, size)
	var newBatch blob.Batch
	for i := 0; i < size; i++ {
		batch[i] = &blob.Object{
			Key:  mdl.Box(fmt.Sprintf("foo-%d", i)),
			Body: blob.StreamBytes([]byte{'f', 'o', 'o'}),
		}
	}

	err = store.Write(batch)
	if !s.NoError(err) {
		return
	}

	newBatch, err = store.ListObjects(s.T().Context(), "")
	if !s.NoError(err) {
		return
	}
	s.Len(newBatch, size)
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
