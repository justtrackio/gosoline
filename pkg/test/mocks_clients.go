package test

import (
	"database/sql"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/go-redis/redis/v8"
)

func (m *Mocks) ProvideSqsClient(name string) *sqs.SQS {
	component := m.components[name].(*snsSqsComponent)
	return component.provideSqsClient()
}

func (m *Mocks) ProvideSnsClient(name string) *sns.SNS {
	component := m.components[name].(*snsSqsComponent)
	return component.provideSnsClient()
}

func (m *Mocks) ProvideCloudwatchClient(name string) *cloudwatch.CloudWatch {
	component := m.components[name].(*cloudwatchComponent)
	return component.provideCloudwatchClient()
}

func (m *Mocks) ProvideDynamoDbClient(name string) *dynamodb.DynamoDB {
	component := m.components[name].(*dynamoDbComponent)
	return component.provideDynamoDbClient()
}

func (m *Mocks) ProvideKinesisClient(name string) *kinesis.Kinesis {
	component := m.components[name].(*kinesisComponent)
	return component.provideKinesisClient()
}

func (m *Mocks) ProvideS3Client(name string) *s3.S3 {
	component := m.components[name].(*s3Component)
	return component.provideS3Client()
}

func (m *Mocks) ProvideMysqlClient(name string) *sql.DB {
	component := m.components[name].(*mysqlComponentLegacy)
	return component.db
}

func (m *Mocks) ProvideRedisClient(name string) *redis.Client {
	component := m.components[name].(*redisComponent)
	return component.provideRedisClient()
}
