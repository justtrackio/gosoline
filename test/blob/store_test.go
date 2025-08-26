//go:build integration

package blob_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type StoreTestSuite struct {
	suite.Suite

	s3Client s3.Client
	stores   map[string]blob.Store
}

func (s *StoreTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("store_test_cfg.yml"),
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(clock.NewRealClock()),
		suite.WithModule("blob-store-runner-default", blob.ProvideBatchRunner("default")),
		suite.WithModule("blob-store-runner-foo", blob.ProvideBatchRunner("foo")),
	}
}

func (s *StoreTestSuite) SetupTest() error {
	var err error

	ctx := s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	s.stores = map[string]blob.Store{}
	var stores map[string]any
	if stores, err = config.GetStringMap("blob"); err != nil {
		return err
	}

	for storeName := range stores {
		if s.stores[storeName], err = blob.NewStore(ctx, config, logger, storeName); err != nil {
			return err
		}
	}

	if s.s3Client, err = s3.ProvideClient(ctx, config, logger, "default"); err != nil {
		return err
	}

	return nil
}

// This tests the following scenarios:
// * general ability to store objects
// * pagination of the ListObjects func of the store
func (s *StoreTestSuite) TestNewDefault(_ suite.AppUnderTest) {
	store := s.stores["default"]

	// make sure this exceeds the default batch size for ListObjectV2 requests, as we are testing the pagination here
	size := 1001

	batch := make(blob.Batch, size)
	var newBatch blob.Batch
	for i := 0; i < size; i++ {
		batch[i] = &blob.Object{
			Key:  mdl.Box(fmt.Sprintf("foo-%d", i)),
			Body: blob.StreamBytes([]byte{'f', 'o', 'o'}),
		}
	}

	err := store.Write(batch)
	if !s.NoError(err) {
		return
	}

	newBatch, err = store.ListObjects(s.T().Context(), "")
	if !s.NoError(err) {
		return
	}

	s.Len(newBatch, size)
}

// This Test is to ensure that we can write to multiple stores and that there are no stalled
// go routines during that due to wrong handling in batch runner channel provider
func (s *StoreTestSuite) TestMultiStoreNoBatchRunnerChannelIssues(_ suite.AppUnderTest) {
	stores, err := s.Env().Config().GetStringMap("blob")
	if !s.NoError(err) {
		return
	}

	for storeName := range stores {
		store := s.stores[storeName]

		ch := make(chan struct{})
		t := time.NewTicker(10 * time.Second)
		var writeSuccess bool

		go func() {
			defer close(ch)

			obj := &blob.Object{
				Key:  mdl.Box("data"),
				Body: blob.StreamBytes(bytes.NewBufferString(storeName).Bytes()),
			}

			err = store.WriteOne(obj)
			writeSuccess = s.NoError(err)
		}()

		select {
		case <-ch:
		case <-t.C:
			s.Fail("timeout")
		}

		if !writeSuccess {
			s.Fail("failed to write object")
		}

		batch, err := store.ListObjects(s.T().Context(), "")
		if !s.NoError(err) {
			return
		}

		s.Len(batch, 1)
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
