package blob

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoS3 "github.com/justtrackio/gosoline/pkg/cloud/aws/s3"
)

type Settings struct {
	cfg.Identity
	BucketId   string
	Bucket     string `cfg:"bucket"`
	Region     string `cfg:"region"`
	ClientName string `cfg:"client_name" default:"default"`
	Prefix     string `cfg:"prefix"`
}

func (s Settings) GetIdentity() cfg.Identity {
	return s.Identity
}

func (s Settings) GetClientName() string {
	return s.ClientName
}

func (s Settings) GetBucketId() string {
	return s.BucketId
}

func getConfigKey(name string) string {
	return fmt.Sprintf("blob.%s", name)
}

func ReadStoreSettings(config cfg.Config, name string) (*Settings, error) {
	var err error
	var s3ClientConfig *gosoS3.ClientConfig

	key := getConfigKey(name)
	settings := &Settings{
		BucketId: name,
	}

	if err := config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("blob.default", ".")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blob store settings for %s: %w", name, err)
	}

	if err := settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad blob store identity from config: %w", err)
	}

	if settings.Bucket == "" {
		if settings.Bucket, err = gosoS3.GetBucketName(config, settings); err != nil {
			return nil, fmt.Errorf("failed to format bucket name: %w", err)
		}
	}

	if settings.Region == "" {
		if s3ClientConfig, err = gosoS3.GetClientConfig(config, settings.ClientName); err != nil {
			return nil, fmt.Errorf("failed to get s3 client config for %s: %w", settings.ClientName, err)
		}

		settings.Region = s3ClientConfig.Settings.Region
	}

	return settings, nil
}
