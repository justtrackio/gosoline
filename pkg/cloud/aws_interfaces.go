package cloud

import (
	"github.com/aws/aws-sdk-go/service/applicationautoscaling/applicationautoscalingiface"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

//go:generate mockery --name ApplicationAutoScalingAPI
type ApplicationAutoScalingAPI interface {
	applicationautoscalingiface.ApplicationAutoScalingAPI
}

//go:generate mockery --name CloudWatchAPI
type CloudWatchAPI interface {
	cloudwatchiface.CloudWatchAPI
}

//go:generate mockery --name DynamoDBAPI
type DynamoDBAPI interface {
	dynamodbiface.DynamoDBAPI
}

//go:generate mockery --name EC2API
type EC2API interface {
	ec2iface.EC2API
}

//go:generate mockery --name ECSAPI
type ECSAPI interface {
	ecsiface.ECSAPI
}

//go:generate mockery --name KinesisAPI
type KinesisAPI interface {
	kinesisiface.KinesisAPI
}

//go:generate mockery --name RDSAPI
type RDSAPI interface {
	rdsiface.RDSAPI
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
