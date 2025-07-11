package metadata

import (
	"context"
	"fmt"
	netHttp "net/http"
	"strconv"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	PathAmiId                                                = "ami-id"
	PathAmiLaunchIndex                                       = "ami-launch-index"
	PathAmiManifestPath                                      = "ami-manifest-path"
	PathBlockDeviceMappingAmi                                = "block-device-mapping/ami"
	PathBlockDeviceMappingEbs2                               = "block-device-mapping/ebs2"
	PathBlockDeviceMappingRoot                               = "block-device-mapping/root"
	PathEventsMaintenanceHistory                             = "events/maintenance/history"
	PathEventsMaintenanceScheduled                           = "events/maintenance/scheduled"
	PathHostname                                             = "hostname"
	PathIamInfo                                              = "iam/info"
	PathIamSecurityCredentialList                            = "iam/security-credentials/"
	PathIamSecurityCredentials                               = "iam/security-credentials/%s"
	PathIdentityCredentialsEC2Info                           = "identity-credentials/ec2/info"
	PathIdentityCredentialsEC2SecurityCredentialsEC2Instance = "identity-credentials/ec2/security-credentials/ec2-instance"
	PathInstanceAction                                       = "instance-action"
	PathInstanceId                                           = "instance-id"
	PathInstanceLifeCycle                                    = "instance-life-cycle"
	PathInstanceType                                         = "instance-type"
	PathLocalHostname                                        = "local-hostname"
	PathLocalIpv4                                            = "local-ipv4"
	PathMac                                                  = "mac"
	PathMetricsVhostmd                                       = "metrics/vhostmd"
	PathNetworkInterfaces                                    = "network/interfaces/macs/"
	PathNetworkInterfacesDeviceNumber                        = "network/interfaces/macs/%s/device-number"
	PathNetworkInterfacesInterfaceId                         = "network/interfaces/macs/%s/interface-id"
	PathNetworkInterfacesLocalHostname                       = "network/interfaces/macs/%s/local-hostname"
	PathNetworkInterfacesLocalIpv4s                          = "network/interfaces/macs/%s/local-ipv4s"
	PathNetworkInterfacesMac                                 = "network/interfaces/macs/%s/mac"
	PathNetworkInterfacesNetworkCard                         = "network/interfaces/macs/%s/network-card"
	PathNetworkInterfacesOwnerId                             = "network/interfaces/macs/%s/owner-id"
	PathNetworkInterfacesSecurityGroupIds                    = "network/interfaces/macs/%s/security-group-ids"
	PathNetworkInterfacesSecurityGroups                      = "network/interfaces/macs/%s/security-groups"
	PathNetworkInterfacesSubnetId                            = "network/interfaces/macs/%s/subnet-id"
	PathNetworkInterfacesSubnetIpv4CidrBlock                 = "network/interfaces/macs/%s/subnet-ipv4-cidr-block"
	PathNetworkInterfacesSubnetIpv6CidrBlocks                = "network/interfaces/macs/%s/subnet-ipv6-cidr-blocks"
	PathNetworkInterfacesVpcId                               = "network/interfaces/macs/%s/vpc-id"
	PathNetworkInterfacesVpcIpv4CidrBlock                    = "network/interfaces/macs/%s/vpc-ipv4-cidr-block"
	PathNetworkInterfacesVpcIpv4CidrBlocks                   = "network/interfaces/macs/%s/vpc-ipv4-cidr-blocks"
	PathNetworkInterfacesVpcIpv6CidrBlocks                   = "network/interfaces/macs/%s/vpc-ipv6-cidr-blocks"
	PathPlacementAvailabilityZone                            = "placement/availability-zone"
	PathPlacementAvailabilityZoneId                          = "placement/availability-zone-id"
	PathPlacementRegion                                      = "placement/region"
	PathProfile                                              = "profile"
	PathReservationId                                        = "reservation-id"
	PathSecurityGroups                                       = "security-groups"
	PathServicesDomain                                       = "services/domain"
	PathServicesPartition                                    = "services/partition"
	PathSystem                                               = "system"
)

// A Provider gives you convenient access to the metadata of your EC2 instance.
// See also https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html.
//
//go:generate go run github.com/vektra/mockery/v2 --name Provider
type Provider interface {
	GetMetadata(ctx context.Context, path string) (string, error)
}

var ErrNotAvailable = fmt.Errorf("metadata not available")

type Settings struct {
	// The Host setting allows you to overwrite the endpoint we query metadata from
	Host string `cfg:"host" default:"169.254.169.254"`
	// If you are not running on a EC2 instance, you can set Available to false, and we won't try to query a non-existing
	// endpoint, but instead return a NotAvailableError. Calling code is expected to handle this error and react accordingly.
	Available bool `cfg:"available" default:"true"`
}

type provider struct {
	httpClient http.Client
	clock      clock.Clock
	settings   Settings

	lck            sync.Mutex
	token          string
	tokenExpiresAt time.Time
}

func ProvideProvider(ctx context.Context, config cfg.Config, logger log.Logger) (Provider, error) {
	return appctx.Provide(ctx, provider{}, func() (Provider, error) {
		return newProvider(ctx, config, logger, "default")
	})
}

func newProvider(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Provider, error) {
	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "metadata-provider")
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
	}

	var settings Settings
	if err := config.UnmarshalKey(fmt.Sprintf("cloud.aws.%s.ec2.metadata", name), &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ec2 metadata settings for %s: %w", name, err)
	}

	return NewProviderWithInterfaces(httpClient, clock.Provider, settings), nil
}

func NewProviderWithInterfaces(httpClient http.Client, clock clock.Clock, settings Settings) Provider {
	return &provider{
		httpClient: httpClient,
		clock:      clock,
		settings:   settings,
	}
}

func (m *provider) GetMetadata(ctx context.Context, path string) (string, error) {
	if !m.settings.Available {
		return "", ErrNotAvailable
	}

	token, err := m.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get new token: %w", err)
	}

	req := m.httpClient.NewRequest().
		WithUrl(fmt.Sprintf("http://%s/latest/meta-data/%s", m.settings.Host, path)).
		WithHeader("X-aws-ec2-metadata-token", token)
	res, err := m.httpClient.Get(ctx, req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != netHttp.StatusOK {
		return "", fmt.Errorf("unexpectd status %d", res.StatusCode)
	}

	return string(res.Body), nil
}

func (m *provider) getToken(ctx context.Context) (string, error) {
	m.lck.Lock()
	defer m.lck.Unlock()

	if m.token != "" && m.tokenExpiresAt.Before(m.clock.Now()) {
		return m.token, nil
	}

	tokenDuration := 21600 * time.Second
	req := m.httpClient.NewRequest().
		WithUrl("http://169.254.169.254/latest/api/token").
		WithHeader("X-aws-ec2-metadata-token-ttl-seconds", strconv.Itoa(int(tokenDuration.Seconds())))
	res, err := m.httpClient.Put(ctx, req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != netHttp.StatusOK {
		return "", fmt.Errorf("unexpectd status %d", res.StatusCode)
	}

	m.token = string(res.Body)
	m.tokenExpiresAt = m.clock.Now().Add(tokenDuration - time.Second*10) // renew shortly before we need to

	return m.token, nil
}
