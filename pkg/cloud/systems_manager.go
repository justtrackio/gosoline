package cloud

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	ssm2 "github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"strings"
)

type SsmParameters map[string]string

//go:generate mockery -name SsmClient
type SsmClient interface {
	GetParameters(string) (SsmParameters, error)
	GetParameter(string) (string, error)
}

type SimpleSystemsManager struct {
	logger log.Logger
	client ssmiface.SSMAPI
}

func NewSimpleSystemsManager(config cfg.Config, logger log.Logger) *SimpleSystemsManager {
	client := GetSystemsManagerClient(config, logger)

	return &SimpleSystemsManager{
		logger: logger,
		client: client,
	}
}

func (ssm SimpleSystemsManager) GetParameters(path string) (SsmParameters, error) {
	input := &ssm2.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}

	out, err := ssm.client.GetParametersByPath(input)

	if err != nil {
		return SsmParameters{}, err
	}

	params := make(SsmParameters)
	for _, p := range out.Parameters {
		key := strings.Replace(*p.Name, path+"/", "", -1)
		params[key] = *p.Value
	}

	return params, nil
}

func (ssm SimpleSystemsManager) GetParameter(path string) (string, error) {
	input := &ssm2.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(true),
	}

	out, err := ssm.client.GetParameter(input)

	if err != nil {
		return "", err
	}

	return *out.Parameter.Value, nil
}
