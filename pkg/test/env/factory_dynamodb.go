package env

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func init() {
	componentFactories[componentDynamoDb] = new(dynamoDbFactory)
}

const componentDynamoDb = "dynamodb"

type dynamoDbSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	ContainerBindingSettings
	Region string `cfg:"region" default:"eu-central-1"`
}

type dynamoDbFactory struct{}

func (f *dynamoDbFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *dynamoDbFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &dynamoDbSettings{}
}

func (f *dynamoDbFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*dynamoDbSettings)
	ports := PortBindings{
		"main": {ContainerPort: 8000, HostPort: s.Port, Protocol: "tcp"},
	}
	containerConfig := &ContainerConfig{
		Auth: s.Image.Auth, Repository: s.Image.Repository, Tag: s.Image.Tag,
		PortBindings: ports,
		Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-inMemory", "-sharedDb", "-disableTelemetry"},
	}
	if s.isExternal() {
		containerConfig = externalContainer(s.Host, ports)
	}

	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: containerConfig,
			HealthCheck: func(container *Container) error {
				component := &dynamoDbComponent{binding: container.bindings["main"], region: s.Region}
				_, err := component.DdbClient().ListTables(context.Background(), &dynamodb.ListTablesInput{})

				return err
			},
		},
	}
}

func (f *dynamoDbFactory) Component(config cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	naming, err := ddb.GetTableNamingSettings(config, "default")
	if err != nil {
		return nil, fmt.Errorf("can not read DynamoDB table naming: %w", err)
	}

	return &dynamoDbComponent{
		binding:      containers["main"].bindings["main"],
		region:       settings.(*dynamoDbSettings).Region,
		tablePattern: "goso-" + uuid.New().NewV4() + "-" + naming.TablePattern,
	}, nil
}
