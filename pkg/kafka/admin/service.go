package admin

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
)

type Service struct {
	logger log.Logger
	client Client
}

func NewService(ctx context.Context, logger log.Logger, brokers []string) (*Service, error) {
	client, err := NewClient(ctx, logger, brokers)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka admin client: %w", err)
	}

	return &Service{
		logger: logger,
		client: client,
	}, nil
}

func (s *Service) CreateTopic(ctx context.Context, topic string) error {
	topicList, err := s.client.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if topicList.Has(topic) {
		return nil
	}

	res, err := s.client.CreateTopic(ctx, 1, 1, nil, topic)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	if res.Err != nil {
		return fmt.Errorf("failed to create topic: %s: %w", res.ErrMessage, res.Err)
	}

	return nil
}
