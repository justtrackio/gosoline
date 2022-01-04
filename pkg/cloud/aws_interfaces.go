package cloud

import (
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

//go:generate mockery --name LambdaApi
type LambdaApi interface {
	lambdaiface.LambdaAPI
}
