package instancemetadataservice

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name InstanceMetadataService
type InstanceMetadataService interface {
	GetAll(ctx context.Context) (map[string]interface{}, error)
	GetIp(ctx context.Context) (string, error)
}

type instanceMetadataService struct {
	client Client
}

func NewInstanceMetadataService(ctx context.Context, config cfg.Config, logger log.Logger) (*instanceMetadataService, error) {
	client, err := ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create imds client: %w", err)
	}

	return &instanceMetadataService{
		client: client,
	}, nil
}

func (s instanceMetadataService) GetAll(ctx context.Context) (map[string]interface{}, error) {
	out, err := s.client.GetMetadata(ctx, &imds.GetMetadataInput{})
	if err != nil {
		return nil, err
	}

	_ = out

	metadata := make(map[string]interface{}, 0)
	//for k, v := range out.ResultMetadata.{
	//	print(k, v)
	//}

	return metadata, nil
}

func (s instanceMetadataService) GetIp(ctx context.Context) (string, error) {
	ip, err := s.client.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "local-ipv4",
	})
	if err != nil {
		return "", err
	}

	print(ip)

	return "", nil
}
