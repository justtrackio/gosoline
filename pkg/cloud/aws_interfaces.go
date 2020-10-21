package cloud

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

//go:generate mockery --name CloudWatchAPI
type CloudWatchAPI interface {
	cloudwatchiface.CloudWatchAPI
}

//go:generate mockery --name DynamoDBAPI
type DynamoDBAPI interface {
	dynamodbiface.DynamoDBAPI
}

//go:generate mockery --name ECSAPI
type ECSAPI interface {
	ecsiface.ECSAPI
}

//go:generate mockery --name KinesisAPI
type KinesisAPI interface {
	kinesisiface.KinesisAPI
}

//go:generate mockery --name ResourceGroupsTaggingAPIAPI
type ResourceGroupsTaggingAPIAPI interface {
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
}

//go:generate mockery --name ServiceDiscoveryAPI
type ServiceDiscoveryAPI interface {
	servicediscoveryiface.ServiceDiscoveryAPI
}

//go:generate mockery --name SSMAPI
type SSMAPI interface {
	ssmiface.SSMAPI
}
