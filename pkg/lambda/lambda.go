package lambda

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	awsLambda "github.com/aws/aws-lambda-go/lambda"
)

type Handler func(config cfg.Config, logger mon.Logger) interface{}

func Start(handler Handler) {
	config := cfg.New()
	logger := mon.NewLogger()

	lambdaHandler := handler(config, logger)
	awsLambda.Start(lambdaHandler)
}
