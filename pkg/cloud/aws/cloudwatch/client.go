package cloudwatch

import (
	"context"
	"fmt"
	"sync"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

//go:generate mockery --name Client
type Client interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type Settings struct {
	gosoAws.ClientSettings
}

var clients = struct {
	lck       sync.Mutex
	instances map[string]*cloudwatch.Client
}{
	instances: map[string]*cloudwatch.Client{},
}

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...func(options *awsCfg.LoadOptions) error) (*cloudwatch.Client, error) {
	clients.lck.Lock()
	defer clients.lck.Unlock()

	var ok bool
	var err error
	var client *cloudwatch.Client

	if client, ok = clients.instances[name]; ok {
		return client, nil
	}

	if client, err = NewClient(ctx, config, logger, name, optFns...); err != nil {
		return nil, err
	}

	clients.instances[name] = client

	return client, nil
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...func(options *awsCfg.LoadOptions) error) (*cloudwatch.Client, error) {
	settings := &Settings{}
	gosoAws.UnmarshalClientSettings(config, settings, "cloudwatch", name)

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, logger, clock.Provider, settings.ClientSettings, optFns...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsConfig)

	return client, nil
}
