package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"strconv"
	"strings"
	"sync"
)

const DefaultVisibilityTimeout = "30"

type ServiceSettings struct {
	AutoCreate bool
}

//go:generate mockery -name Service
type Service interface {
	CreateQueue(*Settings) (*Properties, error)
	QueueExists(string) (bool, error)
	GetPropertiesByName(string) (*Properties, error)
	GetPropertiesByArn(string) (*Properties, error)
	GetUrl(string) (string, error)
	GetArn(string) (string, error)
	Purge(string) error
}

type service struct {
	lck      sync.Mutex
	logger   log.Logger
	client   sqsiface.SQSAPI
	settings *ServiceSettings
}

func NewService(config cfg.Config, logger log.Logger) Service {
	client := ProvideClient(config, logger, &Settings{})
	settings := &ServiceSettings{
		AutoCreate: config.GetBool("aws_sqs_autoCreate"),
	}

	return NewServiceWithInterfaces(logger, client, settings)
}

func NewServiceWithInterfaces(logger log.Logger, client sqsiface.SQSAPI, settings *ServiceSettings) Service {
	return &service{
		logger:   logger,
		client:   client,
		settings: settings,
	}
}

func (s *service) CreateQueue(settings *Settings) (*Properties, error) {
	s.lck.Lock()
	defer s.lck.Unlock()

	name := QueueName(settings)
	exists, err := s.QueueExists(name)

	if err != nil {
		return nil, err
	}

	if exists {
		return s.GetPropertiesByName(name)
	}

	if !exists && !s.settings.AutoCreate {
		return nil, fmt.Errorf("sqs queue with name %s does not exist", name)
	}

	attributes, err := s.createDeadLetterQueue(settings)

	if err != nil {
		return nil, err
	}

	sqsInput := &sqs.CreateQueueInput{
		QueueName:  aws.String(name),
		Attributes: make(map[string]*string),
	}

	for k, v := range attributes {
		sqsInput.Attributes[k] = v
	}

	if settings.Fifo.Enabled {
		sqsInput.Attributes[sqs.QueueAttributeNameFifoQueue] = aws.String("true")
	}

	if settings.Fifo.Enabled && settings.Fifo.ContentBasedDeduplication {
		sqsInput.Attributes[sqs.QueueAttributeNameContentBasedDeduplication] = aws.String("true")
	}

	props, err := s.doCreateQueue(sqsInput)

	if err != nil {
		return nil, err
	}

	visibilityTimeout := DefaultVisibilityTimeout
	if settings.VisibilityTimeout > 0 {
		visibilityTimeout = strconv.Itoa(settings.VisibilityTimeout)
	}

	_, err = s.client.SetQueueAttributes(&sqs.SetQueueAttributesInput{
		QueueUrl: aws.String(props.Url),
		Attributes: map[string]*string{
			sqs.QueueAttributeNameVisibilityTimeout: aws.String(visibilityTimeout),
		},
	})

	return props, err
}

func (s *service) QueueExists(name string) (bool, error) {
	s.logger.WithFields(log.Fields{
		"name": name,
	}).Info("checking the existence of sqs queue")

	url, err := s.GetUrl(name)

	if err != nil {
		return false, err
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

func (s *service) GetPropertiesByArn(arn string) (*Properties, error) {
	i := strings.LastIndex(arn, ":")
	name := arn[i+1:]

	url, err := s.GetUrl(name)

	if err != nil {
		return nil, err
	}

	return &Properties{
		Name: name,
		Url:  url,
		Arn:  arn,
	}, nil
}

func (s *service) GetPropertiesByName(name string) (*Properties, error) {
	url, err := s.GetUrl(name)

	if err != nil {
		return nil, err
	}

	arn, err := s.GetArn(url)

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

func (s *service) GetUrl(name string) (string, error) {
	input := &sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	}

	out, err := s.client.GetQueueUrl(input)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == sqs.ErrCodeQueueDoesNotExist {
			return "", nil
		}

		return "", err
	}

	return *out.QueueUrl, nil
}

func (s *service) GetArn(url string) (string, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String(url),
	}

	out, err := s.client.GetQueueAttributes(input)

	if err != nil {
		return "", err
	}

	arn := *(out.Attributes["QueueArn"])

	return arn, nil
}

func (s *service) Purge(url string) error {
	_, err := s.client.PurgeQueue(&sqs.PurgeQueueInput{
		QueueUrl: aws.String(url),
	})

	return err
}

func (s *service) createDeadLetterQueue(settings *Settings) (map[string]*string, error) {
	attributes := make(map[string]*string)

	if !settings.RedrivePolicy.Enabled {
		return attributes, nil
	}

	queueName := namingStrategy(settings.AppId, settings.QueueId)

	deadLetterName := deadLetterNamingStrategy(settings.AppId, settings.QueueId)
	deadLetterInput := &sqs.CreateQueueInput{
		QueueName: aws.String(deadLetterName),
	}

	props, err := s.doCreateQueue(deadLetterInput)

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
		return attributes, fmt.Errorf("could not marshal redrive policy for sqs queue %v: %w", queueName, err)
	}

	attributes[sqs.QueueAttributeNameRedrivePolicy] = aws.String(string(b))

	return attributes, nil
}

func (s *service) doCreateQueue(input *sqs.CreateQueueInput) (*Properties, error) {
	name := *input.QueueName

	s.logger.Info("trying to create sqs queue: %v", name)
	_, err := s.client.CreateQueue(input)

	if err != nil {
		s.logger.Error("could not create sqs queue %v: %w", name, err)
		return nil, err
	}

	s.logger.Info("created sqs queue %v", name)

	return s.GetPropertiesByName(name)
}
