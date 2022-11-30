package metric

import "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

const (
	PriorityLow  = 0
	PriorityHigh = 1

	UnitCount        = types.StandardUnitCount
	UnitSeconds      = types.StandardUnitSeconds
	UnitMilliseconds = types.StandardUnitMilliseconds
)
