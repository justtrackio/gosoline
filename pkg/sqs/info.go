package sqs

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

var namingStrategy = func(appId cfg.AppId, queueId string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, queueId)
}

func WithNamingStrategy(strategy func(appId cfg.AppId, queueId string) string) {
	namingStrategy = strategy
}

func QueueExists(logger mon.Logger, client sqsiface.SQSAPI, s Settings) bool {
	name := namingStrategy(s.AppId, s.QueueId)
	logger.WithFields(mon.Fields{
		"name": name,
	}).Info("checking the existens of sqs queue")

	input := &sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(name),
	}

	out, err := client.ListQueues(input)

	if err != nil {
		logger.Fatal(err, "could not get the list of sqs queues")
	}

	logger.Info(fmt.Sprintf("found %v queues", len(out.QueueUrls)))

	return len(out.QueueUrls) == 1
}

func GetUrl(logger mon.Logger, client sqsiface.SQSAPI, s Settings) string {
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
		}).Fatal(err, "could not get url of sqs queue")
	}

	logger.WithFields(mon.Fields{
		"name": name,
		"url":  *out.QueueUrl,
	}).Info("found url of sqs queue")

	return *out.QueueUrl
}

func GetArn(logger mon.Logger, client sqsiface.SQSAPI, s Settings) string {
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
		}).Fatal(err, "could not get arn of sqs queue")
	}

	arn := *(out.Attributes["QueueArn"])
	logger.WithFields(mon.Fields{
		"name": name,
		"arn":  arn,
	}).Info("found arn of sqs queue")

	return arn
}
