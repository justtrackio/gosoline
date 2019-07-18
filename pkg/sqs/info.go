package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type NamingFactory func(appId cfg.AppId, queueId string) string

var namingStrategy = func(appId cfg.AppId, queueId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, queueId)
}

func WithNamingStrategy(strategy NamingFactory) {
	namingStrategy = strategy
}

func QueueExists(logger mon.Logger, client sqsiface.SQSAPI, s Settings) bool {
	name := namingStrategy(s.AppId, s.QueueId)
	logger.WithFields(mon.Fields{
		"name": name,
	}).Info("checking the existence of sqs queue")

	url, err := GetUrl(logger, client, s)

	if err != nil {
		logger.Warn(err, "could not get the list of sqs queues")

		return false
	}

	logger.Info(fmt.Sprintf("found queue %s with url %s", name, url))

	return len(url) > 0
}

func GetUrl(logger mon.Logger, client sqsiface.SQSAPI, s Settings) (string, error) {
	name := namingStrategy(s.AppId, s.QueueId)
	logger.WithFields(mon.Fields{
		"name": name,
	}).Info("trying to get the url of sqs queue")

	input := &sqs.GetQueueUrlInput{
		QueueName: aws.String(name),
	}

	out, err := client.GetQueueUrl(input)

	if err != nil {
		logger.WithFields(mon.Fields{
			"name": name,
		}).Warn(err, "could not get url of sqs queue")

		return "", err
	}

	logger.WithFields(mon.Fields{
		"name": name,
		"url":  *out.QueueUrl,
	}).Info("found url of sqs queue")

	return *out.QueueUrl, nil
}

func GetArn(logger mon.Logger, client sqsiface.SQSAPI, s Settings) (string, error) {
	name := namingStrategy(s.AppId, s.QueueId)
	logger.WithFields(mon.Fields{
		"name": name,
	}).Info("trying to get the arn of sqs queue")

	input := &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String(s.Url),
	}

	out, err := client.GetQueueAttributes(input)

	if err != nil {
		logger.WithFields(mon.Fields{
			"name": name,
		}).Warn(err, "could not get arn of sqs queue")
	}

	arn := *(out.Attributes["QueueArn"])
	logger.WithFields(mon.Fields{
		"name": name,
		"arn":  arn,
	}).Info("found arn of sqs queue")

	return arn, nil
}
