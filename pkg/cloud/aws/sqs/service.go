package sqs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	DefaultVisibilityTimeout = "30"
	DeadletterFifoSuffix     = "-dead.fifo"
	FifoSuffix               = ".fifo"
)

//go:generate go run github.com/vektra/mockery/v2 --name Service
type Service interface {
	CreateQueue(ctx context.Context) (*Properties, error)
	QueueExists(ctx context.Context, name string) (bool, error)
	Purge(ctx context.Context) error
	PropertiesResolver
}

type service struct {
	PropertiesResolver
	logger   log.Logger
	client   Client
	settings *Settings
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, optFns ...ClientOption) (*service, error) {
	var err error
	var client Client

	if client, err = ProvideClient(ctx, config, logger, settings.ClientName, optFns...); err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return NewServiceWithInterfaces(logger, client, settings), nil
}

func NewServiceWithInterfaces(logger log.Logger, client Client, settings *Settings) *service {
	return &service{
		PropertiesResolver: NewPropertiesResolverWithInterfaces(client),
		logger:             logger,
		client:             client,
		settings:           settings,
	}
}

func (s *service) CreateQueue(ctx context.Context) (*Properties, error) {
	var err error
	var exists bool

	if exists, err = s.QueueExists(ctx, s.settings.QueueName); err != nil {
		return nil, fmt.Errorf("can not check if queue already exits: %w", err)
	}

	if exists {
		return s.GetPropertiesByName(ctx, s.settings.QueueName)
	}

	attributes, err := s.createDeadLetterQueue(ctx, s.settings)
	if err != nil {
		return nil, err
	}

	sqsInput := &sqs.CreateQueueInput{
		QueueName:  aws.String(s.settings.QueueName),
		Attributes: make(map[string]string),
	}

	for k, v := range attributes {
		sqsInput.Attributes[k] = v
	}

	if s.settings.Fifo.Enabled {
		sqsInput.Attributes[string(types.QueueAttributeNameFifoQueue)] = strconv.FormatBool(true)
	}

	if s.settings.Fifo.Enabled && s.settings.Fifo.ContentBasedDeduplication {
		sqsInput.Attributes[string(types.QueueAttributeNameContentBasedDeduplication)] = strconv.FormatBool(true)
	}

	props, err := s.doCreateQueue(ctx, sqsInput)
	if err != nil {
		return nil, err
	}

	visibilityTimeout := DefaultVisibilityTimeout
	if s.settings.VisibilityTimeout > 0 {
		visibilityTimeout = strconv.Itoa(s.settings.VisibilityTimeout)
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

	if url != "" {
		s.logger.Info("found queue %s with url %s", name, url)

		return true, nil
	}

	s.logger.WithFields(log.Fields{
		"name": name,
	}).Info("could not find queue")

	return false, nil
}

func (s *service) Purge(ctx context.Context) error {
	var err error
	var url string

	if url, err = s.GetUrl(ctx, s.settings.QueueName); err != nil {
		return fmt.Errorf("can not get url of queue: %w", err)
	}

	if _, err = s.client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: aws.String(url)}); err != nil {
		return fmt.Errorf("can not purge queue: %w", err)
	}

	return err
}

func (s *service) createDeadLetterQueue(ctx context.Context, settings *Settings) (map[string]string, error) {
	attributes := make(map[string]string)

	if !s.settings.RedrivePolicy.Enabled {
		return attributes, nil
	}

	deadLetterAttributes := map[string]string{}
	deadLetterName := fmt.Sprintf("%s-dead", s.settings.QueueName)

	if s.settings.Fifo.Enabled {
		deadLetterAttributes[string(types.QueueAttributeNameFifoQueue)] = strconv.FormatBool(true)
		deadLetterName = strings.Replace(s.settings.QueueName, FifoSuffix, DeadletterFifoSuffix, 1)
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
		"maxReceiveCount":     strconv.Itoa(s.settings.RedrivePolicy.MaxReceiveCount),
	}

	b, err := json.Marshal(policy)
	if err != nil {
		return attributes, fmt.Errorf("could not marshal redrive policy for sqs queue %v: %w", s.settings.QueueName, err)
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
