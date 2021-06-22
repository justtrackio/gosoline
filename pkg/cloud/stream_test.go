package cloud_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetStreamClientWithDefault(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "env").Return("environment")
	config.On("GetString", "app_project").Return("project")
	config.On("GetString", "app_family").Return("family")
	config.On("GetString", "app_name").Return("name")
	config.On("GetString", "aws_dynamoDb_endpoint").Return("127.0.0.1")
	config.On("GetString", "aws_kinesis_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)
	logger := logMocks.NewLoggerMockedAll()

	assert.NotPanics(t, func() {
		cloud.GetStreamClientWithDefault(config, logger)
	})
}

func TestStreamClient_GetActiveShardCount(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	dyn := new(cloudMocks.DynamoDBAPI)
	dyn.On("GetItem", &dynamodb.GetItemInput{
		TableName: aws.String("project-environment-family-application2-event_metadata"),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String("ShardCache"),
			},
		},
	}).Return(&dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"ShardIDs": {
				SS: []*string{
					aws.String("bla"),
					aws.String("bla2"),
				},
			},
		},
	}, nil)
	kin := new(cloudMocks.KinesisAPI)

	appId := &cfg.AppId{
		Project:     "project",
		Environment: "environment",
		Family:      "family",
		Application: "application",
	}

	sc := cloud.GetStreamClientWithInterfaces(logger, appId, dyn, kin, "environment")
	count := sc.GetActiveShardCount("application2", "event")

	assert.Equal(t, 2, count)
}

func TestStreamClient_SetShardCount(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	dyn := new(cloudMocks.DynamoDBAPI)

	kin := new(cloudMocks.KinesisAPI)
	kin.On("UpdateShardCount", &kinesis.UpdateShardCountInput{
		ScalingType:      aws.String(kinesis.ScalingTypeUniformScaling),
		StreamName:       aws.String("test"),
		TargetShardCount: aws.Int64(2),
	}).Return(&kinesis.UpdateShardCountOutput{}, nil)

	appId := &cfg.AppId{}

	sc := cloud.GetStreamClientWithInterfaces(logger, appId, dyn, kin, "environment")

	input := &cloud.ScaleStreamInput{
		Streams: []string{
			"test",
		},
		Count: 2,
	}

	streamUpdates := sc.SetShardCount(input)

	assert.Len(t, streamUpdates, 1)
}
