package sqs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/dx"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

const DefaultVisibilityTimeout = "30"

type ServiceSettings struct {
	AutoCreate bool
}

//go:generate mockery --name Service
type Service interface {
	CreateQueue(ctx context.Context, settings *Settings) (*Properties, error)
	QueueExists(ctx context.Context, name string) (bool, error)
	GetPropertiesByName(ctx context.Context, name string) (*Properties, error)
	GetPropertiesByArn(ctx context.Context, arn string) (*Properties, error)
	GetUrl(ctx context.Context, name string) (string, error)
	GetArn(ctx context.Context, url string) (string, error)
	Purge(ctx context.Context, url string) error
}

type service struct {
	lck      sync.Mutex
	logger   log.Logger
	client   Client
	settings *ServiceSettings
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, optFns ...ClientOption) (*service, error) {
	var err error
	var client Client

	if client, err = ProvideClient(ctx, config, logger, "default", optFns...); err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	settings := &ServiceSettings{
		AutoCreate: dx.ShouldAutoCreate(config),
	}

	return NewServiceWithInterfaces(logger, client, settings), nil
}

func NewServiceWithInterfaces(logger log.Logger, client Client, settings *ServiceSettings) *service {
	return &service{
		logger:   logger,
		client:   client,
		settings: settings,
	}
}

func (s *service) CreateQueue(ctx context.Context, settings *Settings) (*Properties, error) {
	s.lck.Lock()
	defer s.lck.Unlock()

	var err error
	var exists bool

	if exists, err = s.QueueExists(ctx, settings.QueueName); err != nil {
		return nil, fmt.Errorf("can not check if queue already exits: %w", err)
	}

	if exists {
		return s.GetPropertiesByName(ctx, settings.QueueName)
	}

	if !exists && !s.settings.AutoCreate {
		return nil, fmt.Errorf("sqs queue with name %s does not exist", settings.QueueName)
	}

	attributes, err := s.createDeadLetterQueue(ctx, settings)
	if err != nil {
		return nil, err
	}

	sqsInput := &sqs.CreateQueueInput{
		QueueName:  aws.String(settings.QueueName),
		Attributes: make(map[string]string),
	}

	for k, v := range attributes {
		sqsInput.Attributes[k] = v
	}

	if settings.Fifo.Enabled {
		sqsInput.Attributes[string(types.QueueAttributeNameFifoQueue)] = "true"
	}

	if settings.Fifo.Enabled && settings.Fifo.ContentBasedDeduplication {
		sqsInput.Attributes[string(types.QueueAttributeNameContentBasedDeduplication)] = "true"
	}

	props, err := s.doCreateQueue(ctx, sqsInput)
	if err != nil {
		return nil, err
	}

	visibilityTimeout := DefaultVisibilityTimeout
	if settings.VisibilityTimeout > 0 {
		visibilityTimeout = strconv.Itoa(settings.VisibilityTimeout)
	}

	_, err = s.client.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(props.Url),
		Attributes: map[string]string{
			string(types.QueueAttributeNameVisibilityTimeout): visibilityTimeout,
		},
	})

	return props, err
}

func (s *service) QueueExists(ctx context.Context, name string) (bool, error) {
	s.logger.WithFields(log.Fields{
		"name": name,
	}).Info("checking the existence of sqs queue")

	var err error
	var url string

	if url, err = s.GetUrl(ctx, name); err != nil {
		return false, fmt.Errorf("can not get url of queue: %w", err)
	}

	if len(url) > 0 {
		s.logger.Info("found queue %s with url %s", name, url)
		return true, nil
	}

	s.logger.WithFields(log.Fields{
		"name": name,
	}).Info("could not find queue")

	return false, nil
}

func (s *service) GetPropertiesByArn(ctx context.Context, arn string) (*Properties, error) {
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

func (s *service) GetPropertiesByName(ctx context.Context, name string) (*Properties, error) {
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

func (s *service) GetUrl(ctx context.Context, name string) (string, error) {
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

func (s *service) GetArn(ctx context.Context, url string) (string, error) {
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

func (s *service) Purge(ctx context.Context, url string) error {
	_, err := s.client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: aws.String(url),
	})

	return err
}

func (s *service) createDeadLetterQueue(ctx context.Context, settings *Settings) (map[string]string, error) {
	attributes := make(map[string]string)

	if !settings.RedrivePolicy.Enabled {
		return attributes, nil
	}

	deadLetterAttributes := map[string]string{}
	deadLetterName := fmt.Sprintf("%s-dead", settings.QueueName)

	if settings.Fifo.Enabled {
		deadLetterAttributes[string(types.QueueAttributeNameFifoQueue)] = "true"
		deadLetterName = strings.Replace(settings.QueueName, fifoSuffix, deadletterFifoSuffix, 1)
	}

	deadLetterInput := &sqs.CreateQueueInput{
		Attributes: deadLetterAttributes,
		QueueName:  aws.String(deadLetterName),
	}

	props, err := s.doCreateQueue(ctx, deadLetterInput)
	if err != nil {
		s.logger.Error("could not get arn of dead letter sqs queue %v: %w", deadLetterName, err)
		return attributes, err
	}

	policy := map[string]string{
		"deadLetterTargetArn": props.Arn,
		"maxReceiveCount":     strconv.Itoa(settings.RedrivePolicy.MaxReceiveCount),
	}

	b, err := json.Marshal(policy)
	if err != nil {
		return attributes, fmt.Errorf("could not marshal redrive policy for sqs queue %v: %w", settings.QueueName, err)
	}

	attributes[string(types.QueueAttributeNameRedrivePolicy)] = string(b)

	return attributes, nil
}

func (s *service) doCreateQueue(ctx context.Context, input *sqs.CreateQueueInput) (*Properties, error) {
	name := *input.QueueName
	s.logger.Info("trying to create sqs queue: %v", name)

	if _, err := s.client.CreateQueue(ctx, input); err != nil {
		s.logger.Error("could not create sqs queue %v: %w", name, err)
		return nil, err
	}

	s.logger.Info("created sqs queue %v", name)

	return s.GetPropertiesByName(ctx, name)
}
