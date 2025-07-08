package sqs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name PropertiesResolver
type PropertiesResolver interface {
	GetPropertiesByName(ctx context.Context, name string) (*Properties, error)
	GetPropertiesByArn(ctx context.Context, arn string) (*Properties, error)
	GetUrl(ctx context.Context, name string) (string, error)
	GetArn(ctx context.Context, url string) (string, error)
}

type propertiesResolver struct {
	client Client
}

func NewPropertiesResolver(ctx context.Context, config cfg.Config, logger log.Logger, clientName string, optFns ...ClientOption) (*propertiesResolver, error) {
	var err error
	var client Client

	if client, err = ProvideClient(ctx, config, logger, clientName, optFns...); err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return NewPropertiesResolverWithInterfaces(client), nil
}

func NewPropertiesResolverWithInterfaces(client Client) *propertiesResolver {
	return &propertiesResolver{
		client: client,
	}
}

func (s *propertiesResolver) GetPropertiesByArn(ctx context.Context, arn string) (*Properties, error) {
	i := strings.LastIndex(arn, ":")
	name := arn[i+1:]

	var err error
	var url string

	if url, err = s.GetUrl(ctx, name); err != nil {
		return nil, fmt.Errorf("can not get url: %w", err)
	}

	return &Properties{
		Name: name,
		Url:  url,
		Arn:  arn,
	}, nil
}

func (s *propertiesResolver) GetPropertiesByName(ctx context.Context, name string) (*Properties, error) {
	url, err := s.GetUrl(ctx, name)
	if err != nil {
		return nil, err
	}

	arn, err := s.GetArn(ctx, url)
	if err != nil {
		return nil, err
	}

	properties := &Properties{
		Name: name,
		Url:  url,
		Arn:  arn,
	}

	return properties, nil
}

func (s *propertiesResolver) GetUrl(ctx context.Context, name string) (string, error) {
	var err error
	var out *sqs.GetQueueUrlOutput

	input := &sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	}

	if out, err = s.client.GetQueueUrl(ctx, input); err != nil {
		var errQueueDoesNotExist *types.QueueDoesNotExist
		if errors.As(err, &errQueueDoesNotExist) {
			return "", nil
		}

		return "", fmt.Errorf("can not request queue url: %w", err)
	}

	return *out.QueueUrl, nil
}

func (s *propertiesResolver) GetArn(ctx context.Context, url string) (string, error) {
	var err error
	var out *sqs.GetQueueAttributesOutput

	input := &sqs.GetQueueAttributesInput{
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
		QueueUrl:       aws.String(url),
	}

	if out, err = s.client.GetQueueAttributes(ctx, input); err != nil {
		return "", fmt.Errorf("can not get queue attributes: %w", err)
	}

	arn := out.Attributes["QueueArn"]

	return arn, nil
}
