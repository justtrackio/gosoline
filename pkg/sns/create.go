package sns

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
)

var namingStrategy = func(appId cfg.AppId, topicId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, topicId)
}

func WithNamingStrategy(strategy func(appId cfg.AppId, topicId string) string) {
	namingStrategy = strategy
}

func CreateTopic(logger log.Logger, client snsiface.SNSAPI, s *Settings) (string, error) {
	name := namingStrategy(s.AppId, s.TopicId)

	logger.WithFields(log.Fields{
		"name": name,
	}).Info("looking for sns topic")

	input := &sns.CreateTopicInput{
		Name: aws.String(name),
	}
	out, err := client.CreateTopic(input)

	if err != nil {
		return "", err
	}

	logger.WithFields(log.Fields{
		"name": name,
		"arn":  *out.TopicArn,
	}).Info("found sns topic")

	return *out.TopicArn, nil
}
