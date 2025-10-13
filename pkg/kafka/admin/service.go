package admin

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
)

type Service struct {
	logger log.Logger
	client Client
	topic  string
}

func NewService(ctx context.Context, logger log.Logger, topic string, brokers []string) (*Service, error) {
	client, err := NewClient(ctx, logger, brokers)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka admin client: %w", err)
	}

	return &Service{
		logger: logger,
		client: client,
		topic:  topic,
	}, nil
}

func (s *Service) CreateTopic(ctx context.Context) error {
	topicList, err := s.client.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if topicList.Has(s.topic) {
		return nil
	}

	res, err := s.client.CreateTopic(ctx, 1, 1, nil, s.topic)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	if res.Err != nil {
		return fmt.Errorf("failed to create topic: %s: %w", res.ErrMessage, res.Err)
	}

	return nil
}
