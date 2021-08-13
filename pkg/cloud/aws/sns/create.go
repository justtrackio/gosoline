package sns

import (
	"context"

	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func CreateTopic(ctx context.Context, logger log.Logger, client Client, topicName string) (string, error) {
	logger.WithFields(log.Fields{
		"name": topicName,
	}).Info("looking for sns topic")

	input := &sns.CreateTopicInput{
		Name: aws.String(topicName),
	}

	var err error
	var out *sns.CreateTopicOutput

	if out, err = client.CreateTopic(ctx, input); err != nil {
		return "", err
	}

	logger.WithFields(log.Fields{
		"name": topicName,
		"arn":  *out.TopicArn,
	}).Info("found sns topic")

	return *out.TopicArn, nil
}
