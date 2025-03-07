package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type LifeCyclePurger struct {
	client   Client
	queueUrl string
}

func NewLifeCyclePurger(ctx context.Context, config cfg.Config, logger log.Logger, clientName string, queueUrl string) (*LifeCyclePurger, error) {
	var err error
	var client Client

	if client, err = ProvideClient(ctx, config, logger, clientName); err != nil {
		return nil, fmt.Errorf("can not create dynamodb client: %w", err)
	}

	return &LifeCyclePurger{
		client:   client,
		queueUrl: queueUrl,
	}, err
}

func (p LifeCyclePurger) Purge(ctx context.Context) error {
	if _, err := p.client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: aws.String(p.queueUrl)}); err != nil {
		return fmt.Errorf("can not purge queue: %w", err)
	}

	return nil
}
