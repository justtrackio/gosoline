package topology_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/ec2/metadata"
	metadataMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/ec2/metadata/mocks"
	"github.com/justtrackio/gosoline/pkg/cloud/topology"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

func TestTopologyProviderSuite(t *testing.T) {
	suite.Run(t, new(TopologyProviderSuite))
}

type TopologyProviderSuite struct {
	suite.Suite
	logger *logMocks.Logger
}

func (s *TopologyProviderSuite) SetupTest() {
	s.logger = new(logMocks.Logger)
}

func (s *TopologyProviderSuite) TestEc2Provider() {
	metadataProvider := metadataMocks.NewProvider(s.T())
	metadataProvider.EXPECT().
		GetMetadata(context.Background(), metadata.PathPlacementAvailabilityZone).
		Return("eu-west-1a", nil)

	s.logger.On("Info", context.Background(), "resolved ec2 topology zone: %s", "eu-west-1a").Return()

	provider, err := topology.NewEc2ProviderWithInterfaces(context.Background(), s.logger, metadataProvider)
	s.Require().NoError(err)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Equal("eu-west-1a", zone)
}

func (s *TopologyProviderSuite) TestEc2ProviderTrimsWhitespace() {
	metadataProvider := metadataMocks.NewProvider(s.T())
	metadataProvider.EXPECT().
		GetMetadata(context.Background(), metadata.PathPlacementAvailabilityZone).
		Return("  us-east-1b  \n", nil)

	s.logger.On("Info", context.Background(), "resolved ec2 topology zone: %s", "us-east-1b").Return()

	provider, err := topology.NewEc2ProviderWithInterfaces(context.Background(), s.logger, metadataProvider)
	s.Require().NoError(err)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Equal("us-east-1b", zone)
}

func (s *TopologyProviderSuite) TestEc2ProviderMetadataError() {
	metadataProvider := metadataMocks.NewProvider(s.T())
	metadataProvider.EXPECT().
		GetMetadata(context.Background(), metadata.PathPlacementAvailabilityZone).
		Return("", metadata.ErrNotAvailable)

	_, err := topology.NewEc2ProviderWithInterfaces(context.Background(), s.logger, metadataProvider)
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to get availability zone from ec2 metadata")
}

func (s *TopologyProviderSuite) TestEc2ProviderEmptyZone() {
	metadataProvider := metadataMocks.NewProvider(s.T())
	metadataProvider.EXPECT().
		GetMetadata(context.Background(), metadata.PathPlacementAvailabilityZone).
		Return("  ", nil)

	_, err := topology.NewEc2ProviderWithInterfaces(context.Background(), s.logger, metadataProvider)
	s.Require().Error(err)
	s.Contains(err.Error(), "empty availability zone")
}

func (s *TopologyProviderSuite) TestStaticProvider() {
	settings := topology.Settings{
		Provider: "static",
		Static: topology.StaticSettings{
			Zone: "us-east-1b",
		},
	}

	provider, err := topology.NewStaticProviderWithInterfaces(settings)
	s.Require().NoError(err)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Equal("us-east-1b", zone)
}

func (s *TopologyProviderSuite) TestStaticProviderEmptyZone() {
	settings := topology.Settings{
		Provider: "static",
		Static: topology.StaticSettings{
			Zone: "",
		},
	}

	_, err := topology.NewStaticProviderWithInterfaces(settings)
	s.Require().Error(err)
	s.Contains(err.Error(), "static topology zone is not configured")
}

func (s *TopologyProviderSuite) TestCustomProviderFactory() {
	called := false
	customFactory := func(_ context.Context, _ cfg.Config, _ log.Logger, _ topology.Settings) (topology.Provider, error) {
		called = true

		return &staticTestProvider{zone: "custom-zone"}, nil
	}

	topology.SetProviderFactory("custom-test", customFactory)
	defer topology.SetProviderFactory("custom-test", nil)

	config := cfg.New(map[string]any{
		"cloud": map[string]any{
			"topology": map[string]any{
				"provider": "custom-test",
			},
		},
	})

	provider, err := topology.NewProvider(context.Background(), config, s.logger)
	s.Require().NoError(err)
	s.True(called)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Equal("custom-zone", zone)
}

func (s *TopologyProviderSuite) TestUnknownProvider() {
	config := cfg.New(map[string]any{
		"cloud": map[string]any{
			"topology": map[string]any{
				"provider": "unknown",
			},
		},
	})

	_, err := topology.NewProvider(context.Background(), config, s.logger)
	s.Require().Error(err)
	s.Contains(err.Error(), "unknown topology provider")
}

func (s *TopologyProviderSuite) TestNoProviderConfiguredReturnsNoop() {
	config := cfg.New(map[string]any{
		"cloud": map[string]any{
			"topology": map[string]any{
				"provider": "",
			},
		},
	})

	provider, err := topology.NewProvider(context.Background(), config, s.logger)
	s.Require().NoError(err)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Empty(zone)
}

func (s *TopologyProviderSuite) TestProviderFromConfigWithStatic() {
	config := cfg.New(map[string]any{
		"cloud": map[string]any{
			"topology": map[string]any{
				"provider": "static",
				"static": map[string]any{
					"zone": "ap-southeast-1a",
				},
			},
		},
	})

	provider, err := topology.NewProvider(context.Background(), config, s.logger)
	s.Require().NoError(err)

	zone, err := provider.GetZone(context.Background())
	s.Require().NoError(err)
	s.Equal("ap-southeast-1a", zone)
}

// staticTestProvider is a simple Provider for testing custom factory registration.
type staticTestProvider struct {
	zone string
}

func (p *staticTestProvider) GetZone(_ context.Context) (string, error) {
	return p.zone, nil
}
