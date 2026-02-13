package s3

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type BucketNameSettingsAware interface {
	GetIdentity() cfg.Identity
	GetClientName() string
	GetBucketId() string
}

type BucketNameSettings struct {
	Identity   cfg.Identity
	ClientName string
	BucketId   string
}

func (s BucketNameSettings) GetIdentity() cfg.Identity {
	return s.Identity
}

func (s BucketNameSettings) GetClientName() string {
	return s.ClientName
}

func (s BucketNameSettings) GetBucketId() string {
	return s.BucketId
}

type BucketNamingSettings struct {
	BucketPattern string `cfg:"bucket_pattern,nodecode" default:"{app.namespace}"`
	Delimiter     string `cfg:"delimiter" default:"-"`
}

func GetBucketName(config cfg.Config, settings BucketNameSettingsAware) (string, error) {
	if settings.GetClientName() == "" {
		return "", fmt.Errorf("the client name shouldn't be empty")
	}

	namingKey := fmt.Sprintf("%s.naming", aws.GetClientConfigKey("s3", settings.GetClientName()))
	defaultPatternKey := fmt.Sprintf("%s.naming.bucket_pattern", aws.GetClientConfigKey("s3", "default"))

	namingSettings := &BucketNamingSettings{}
	if err := config.UnmarshalKey(namingKey, namingSettings, cfg.UnmarshalWithDefaultsFromKey(defaultPatternKey, "bucket_pattern")); err != nil {
		return "", fmt.Errorf("failed to unmarshal s3 naming settings for %s: %w", namingKey, err)
	}

	identity := settings.GetIdentity()
	if err := identity.PadFromConfig(config); err != nil {
		return "", fmt.Errorf("failed to pad app identity from config: %w", err)
	}

	name, err := identity.Format(namingSettings.BucketPattern, namingSettings.Delimiter, map[string]string{
		"bucketId": settings.GetBucketId(),
	})
	if err != nil {
		return "", fmt.Errorf("s3 bucket naming failed: %w", err)
	}

	return name, nil
}
