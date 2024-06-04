//go:build integration

package metadata_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cloud/aws/ec2/metadata"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ProviderTestSuite struct {
	suite.Suite
	ctx      context.Context
	provider metadata.Provider
}

func (s *ProviderTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("client_test_cfg.yml"),
	}
}

func (s *ProviderTestSuite) SetupTest() error {
	var err error

	s.ctx = s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	s.provider, err = metadata.ProvideProvider(s.ctx, config, logger)

	return err
}

func (s *ProviderTestSuite) TestNotAvailableInTestSuite() {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()

	result, err := s.provider.GetMetadata(ctx, metadata.PathPlacementAvailabilityZone)

	s.True(errors.Is(err, metadata.ErrNotAvailable))
	s.Equal("", result)
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
