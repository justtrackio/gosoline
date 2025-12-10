//go:build integration

package blob_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite
}

func (s *StoreTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("store_test_cfg.yml"),
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(clock.NewRealClock()),
		// Add batch runners first - they create the channels in appctx
		suite.WithModule("blob-store-runner-default", blob.ProvideBatchRunner("default")),
		suite.WithModule("blob-store-runner-foo", blob.ProvideBatchRunner("foo")),
		// Add store setup module after batch runners - it creates stores that share channels
		// and registers their lifecycle handlers for bucket creation
		suite.WithModule("blob-store-setup", provideStoreSetup("default", "foo")),
	}
}

// provideStoreSetup returns a module factory that creates blob stores during app initialization.
// This ensures stores are created:
// 1. AFTER batch runner modules (so they share the same channels via appctx)
// 2. BEFORE the lifecycle middleware runs (so buckets get created)
// The module itself does nothing - it just triggers store creation as a side effect.
func provideStoreSetup(storeNames ...string) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		for _, name := range storeNames {
			if _, err := blob.ProvideStore(ctx, config, logger, name); err != nil {
				return nil, fmt.Errorf("can not create blob store %s: %w", name, err)
			}
		}

		// Return a no-op background module that just waits for context cancellation
		return &storeSetupModule{}, nil
	}
}

// storeSetupModule is a no-op background module that exists just to trigger store creation
// during the module factory phase.
type storeSetupModule struct {
	kernel.BackgroundModule
}

func (m *storeSetupModule) Run(ctx context.Context) error {
	<-ctx.Done()

	return nil
}

// getStores retrieves blob stores from the app's context.
// Stores were created during app initialization by the store setup module.
func (s *StoreTestSuite) getStores(app suite.AppUnderTest) (map[string]blob.Store, error) {
	ctx := app.Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	stores := map[string]blob.Store{}
	var storeNames map[string]any
	var err error

	if storeNames, err = config.GetStringMap("blob"); err != nil {
		return nil, err
	}

	for storeName := range storeNames {
		// ProvideStore retrieves the existing store from appctx (created by store setup module)
		if stores[storeName], err = blob.ProvideStore(ctx, config, logger, storeName); err != nil {
			return nil, err
		}
	}

	return stores, nil
}

// This tests the following scenarios:
// * general ability to store objects
// * pagination of the ListObjects func of the store
func (s *StoreTestSuite) TestNewDefault(app suite.AppUnderTest) {
	stores, err := s.getStores(app)
	if !s.NoError(err) {
		return
	}

	store := stores["default"]

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

// This Test is to ensure that we can write to multiple stores and that there are no stalled
// go routines during that due to wrong handling in batch runner channel provider
func (s *StoreTestSuite) TestMultiStoreNoBatchRunnerChannelIssues(app suite.AppUnderTest) {
	stores, err := s.getStores(app)
	if !s.NoError(err) {
		return
	}

	for storeName := range stores {
		store := stores[storeName]

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
