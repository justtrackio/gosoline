package ssm

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type SsmParameters map[string]string

//go:generate mockery --name SimpleSystemsManager
type SimpleSystemsManager interface {
	GetParameters(ctx context.Context, path string) (SsmParameters, error)
	GetParameter(ctx context.Context, path string) (string, error)
}

type simpleSystemsManager struct {
	logger log.Logger
	client Client
}

func NewSimpleSystemsManager(ctx context.Context, config cfg.Config, logger log.Logger, clientName string) (*simpleSystemsManager, error) {
	client, err := ProvideClient(ctx, config, logger, clientName)
	if err != nil {
		return nil, fmt.Errorf("can not create ssm client: %w", err)
	}

	return &simpleSystemsManager{
		logger: logger,
		client: client,
	}, nil
}

func (m simpleSystemsManager) GetParameters(ctx context.Context, path string) (SsmParameters, error) {
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      mdl.Box(true),
		WithDecryption: mdl.Box(true),
	}

	out, err := m.client.GetParametersByPath(ctx, input)
	if err != nil {
		return SsmParameters{}, err
	}

	params := make(SsmParameters)
	for _, p := range out.Parameters {
		key := strings.ReplaceAll(*p.Name, path+"/", "")
		params[key] = *p.Value
	}

	return params, nil
}

func (m simpleSystemsManager) GetParameter(ctx context.Context, path string) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: mdl.Box(true),
	}

	out, err := m.client.GetParameter(ctx, input)
	if err != nil {
		return "", err
	}

	return *out.Parameter.Value, nil
}
