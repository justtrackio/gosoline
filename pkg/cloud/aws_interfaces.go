package cloud

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

//go:generate mockery --name DynamoDBAPI
type DynamoDBAPI interface {
	dynamodbiface.DynamoDBAPI
}

//go:generate mockery --name KinesisAPI
type KinesisAPI interface {
	kinesisiface.KinesisAPI
}

//go:generate mockery --name LambdaApi
type LambdaApi interface {
	lambdaiface.LambdaAPI
}
