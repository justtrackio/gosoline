package topology

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/ec2/metadata"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ec2Provider struct {
	zone string
}

func newEc2Provider(ctx context.Context, config cfg.Config, logger log.Logger, _ Settings) (Provider, error) {
	metadataProvider, err := metadata.ProvideProvider(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ec2 metadata provider: %w", err)
	}

	return NewEc2ProviderWithInterfaces(ctx, logger, metadataProvider)
}

// NewEc2ProviderWithInterfaces creates an EC2 topology provider with explicit dependencies.
// This is useful for testing without a full application context.
func NewEc2ProviderWithInterfaces(ctx context.Context, logger log.Logger, metadataProvider metadata.Provider) (Provider, error) {
	zone, err := metadataProvider.GetMetadata(ctx, metadata.PathPlacementAvailabilityZone)
	if err != nil {
		return nil, fmt.Errorf("failed to get availability zone from ec2 metadata: %w", err)
	}

	zone = strings.TrimSpace(zone)
	if zone == "" {
		return nil, fmt.Errorf("ec2 metadata returned an empty availability zone")
	}

	logger.Info(ctx, "resolved ec2 topology zone: %s", zone)

	return &ec2Provider{
		zone: zone,
	}, nil
}

func (p *ec2Provider) GetZone(_ context.Context) (string, error) {
	return p.zone, nil
}
