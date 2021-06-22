package cloud

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

type ShardCache struct {
	Key           string
	LastUpdate    float64
	LastUpdateRFC string
	ShardIDs      []string
}

//go:generate mockery -name StreamClient
type StreamClient interface {
	GetActiveShardCount(application string, eventType string) int
	SetShardCount(input *ScaleStreamInput) []*kinesis.UpdateShardCountOutput
}

type AwsStreamClient struct {
	logger         log.Logger
	appId          *cfg.AppId
	environment    string
	dynamoDbClient dynamodbiface.DynamoDBAPI
	kinesisClient  kinesisiface.KinesisAPI
}

type ActiveShardInput struct {
	Application string
	EventType   string
}

type ScaleStreamInput struct {
	Streams []string
	Count   int
}

func GetStreamClientWithDefault(config cfg.Config, logger log.Logger) StreamClient {
	env := config.GetString("env")

	appId := &cfg.AppId{}
	appId.PadFromConfig(config)

	dyn := GetDynamoDbClient(config, logger)
	kin := GetKinesisClient(config, logger)

	return GetStreamClientWithInterfaces(logger, appId, dyn, kin, env)
}

func GetStreamClientWithInterfaces(logger log.Logger, appId *cfg.AppId, dyn dynamodbiface.DynamoDBAPI, kin kinesisiface.KinesisAPI, env string) StreamClient {
	return &AwsStreamClient{
		logger:         logger,
		appId:          appId,
		environment:    env,
		dynamoDbClient: dyn,
		kinesisClient:  kin,
	}
}

func (sc *AwsStreamClient) GetActiveShardCount(application, eventType string) int {
	tableName := fmt.Sprintf("%s-%s-%s-%s-%s_metadata", sc.appId.Project, sc.appId.Environment, sc.appId.Family, application, eventType)

	out, err := sc.dynamoDbClient.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String("ShardCache"),
			},
		},
	})

	if err != nil {
		sc.logger.WithFields(log.Fields{
			"tableName": tableName,
		}).Warn(err.Error())

		return 1
	}

	shardCache := ShardCache{}

	err = dynamodbattribute.UnmarshalMap(out.Item, &shardCache)

	if err != nil {
		sc.logger.WithFields(log.Fields{
			"tableName": tableName,
		}).Warn("Error unmarshalling dynamodbattribute map: %s", err)

		return 1
	}

	return len(shardCache.ShardIDs)
}

func (sc *AwsStreamClient) SetShardCount(input *ScaleStreamInput) []*kinesis.UpdateShardCountOutput {
	updates := make([]*kinesis.UpdateShardCountOutput, 0, 3)

	for _, stream := range input.Streams {
		input := kinesis.UpdateShardCountInput{
			ScalingType:      aws.String(kinesis.ScalingTypeUniformScaling),
			StreamName:       &stream,
			TargetShardCount: aws.Int64(int64(input.Count)),
		}

		out, err := sc.kinesisClient.UpdateShardCount(&input)

		if err != nil {
			sc.logger.Warn(err.Error())
			continue
		}

		updates = append(updates, out)
	}

	return updates
}
