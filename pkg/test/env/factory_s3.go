package env

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func init() {
	componentFactories[componentS3] = new(s3Factory)
}

const componentS3 = "s3"

type s3Settings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	ContainerBindingSettings
	Region string `cfg:"region" default:"eu-central-1"`
}

type s3Factory struct{}

func (f *s3Factory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *s3Factory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &s3Settings{}
}

func (f *s3Factory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*s3Settings)
	ports := PortBindings{
		"main": {ContainerPort: 5000, HostPort: s.Port, Protocol: "tcp"},
	}
	containerConfig := &ContainerConfig{
		Auth: s.Image.Auth, Repository: s.Image.Repository, Tag: s.Image.Tag,
		PortBindings: ports,
		Env:          map[string]string{"S3_IGNORE_SUBDOMAIN_BUCKETNAME": "true"},
	}
	if s.isExternal() {
		containerConfig = externalContainer(s.Host, ports)
	}

	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: containerConfig,
			HealthCheck: func(container *Container) error {
				component := &s3Component{binding: container.bindings["main"], region: s.Region}
				_, err := component.S3Client().ListBuckets(context.Background(), &s3.ListBucketsInput{})

				return err
			},
		},
	}
}

func (f *s3Factory) Component(config cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	component := &s3Component{
		binding: containers["main"].bindings["main"],
		region:  settings.(*s3Settings).Region,
		buckets: make(map[string]string),
	}
	if !config.IsSet("blob") {
		return component, nil
	}
	stores, err := config.GetStringMap("blob")
	if err != nil {
		return nil, fmt.Errorf("can not read blob stores: %w", err)
	}
	// Keep stores sharing a bucket together, while isolating separate environments.
	suffix := strings.ReplaceAll(uuid.New().NewV4(), "-", "")[:12]
	for name := range stores {
		store, err := blob.ReadStoreSettings(config, name)
		if err != nil {
			return nil, fmt.Errorf("can not read blob store %s: %w", name, err)
		}
		bucket := store.Bucket[:min(len(store.Bucket), 50)]
		if len(store.Bucket) > 50 {
			digest := sha256.Sum256([]byte(store.Bucket))
			bucket = fmt.Sprintf("%s-%x", bucket[:41], digest[:4])
		}
		component.buckets[name] = strings.TrimRight(bucket, "-.") + "-" + suffix
	}

	return component, nil
}
