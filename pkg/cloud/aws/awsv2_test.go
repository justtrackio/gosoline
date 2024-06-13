package aws_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoffDelayer(t *testing.T) {
	initialInterval := time.Millisecond * 100
	maxInerval := time.Minute

	delayer := aws.NewBackoffDelayer(initialInterval, maxInerval)

	i := 1
	last := time.Duration(0)

	for ; i <= 100; i++ {
		delay, _ := delayer.BackoffDelay(i, nil)
		fmt.Printf("%02d: %s\n", i, delay)

		assert.True(t, delay > 0)
		assert.True(t, delay <= maxInerval)

		if delay == maxInerval && delay == last {
			break
		}

		last = delay
	}

	assert.True(t, i < 100)
}

func TestUnmarshalClientSettings(t *testing.T) {
	config := cfg.New()

	err := config.Option(cfg.WithConfigMap(map[string]interface{}{
		"cloud.aws": map[string]interface{}{
			"credentials": map[string]interface{}{
				"access_key_id":     "access key id",
				"secret_access_key": "secret access key",
				"session_token":     "session token",
			},
			"defaults": map[string]interface{}{
				"http_client.timeout": "1s",
				"assume_role":         "role",
			},
			"cloudwatch.clients.metrics.http_client": map[string]interface{}{
				"timeout": "2s",
			},
		},
	}))
	assert.NoError(t, err)

	settings := &aws.ClientSettings{}
	aws.UnmarshalClientSettings(config, settings, "cloudwatch", "default")
	assert.Equal(t, &aws.ClientSettings{
		Region:     "eu-central-1",
		Endpoint:   "http://localhost:4566",
		AssumeRole: "role",
		Credentials: aws.Credentials{
			AccessKeyID:     "access key id",
			SecretAccessKey: "secret access key",
			SessionToken:    "session token",
		},
		HttpClient: aws.ClientHttpSettings{
			Timeout: time.Second,
		},
		Backoff: exec.BackoffSettings{
			CancelDelay:     0,
			InitialInterval: time.Millisecond * 50,
			MaxAttempts:     10,
			MaxElapsedTime:  time.Minute * 10,
			MaxInterval:     time.Second * 10,
		},
	}, settings)

	settings = &aws.ClientSettings{}
	aws.UnmarshalClientSettings(config, settings, "cloudwatch", "metrics")
	assert.Equal(t, &aws.ClientSettings{
		Region:     "eu-central-1",
		Endpoint:   "http://localhost:4566",
		AssumeRole: "role",
		Credentials: aws.Credentials{
			AccessKeyID:     "access key id",
			SecretAccessKey: "secret access key",
			SessionToken:    "session token",
		},
		HttpClient: aws.ClientHttpSettings{
			Timeout: time.Second * 2,
		},
		Backoff: exec.BackoffSettings{
			CancelDelay:     0,
			InitialInterval: time.Millisecond * 50,
			MaxAttempts:     10,
			MaxElapsedTime:  time.Minute * 10,
			MaxInterval:     time.Second * 10,
		},
	}, settings)
}
