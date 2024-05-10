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
		delay, err := delayer.BackoffDelay(i, nil)
		assert.NoError(t, err)

		_, err = fmt.Printf("%02d: %s\n", i, delay)
		assert.NoError(t, err)

		assert.True(t, delay > 0)
		assert.True(t, delay <= maxInerval)

		if delay == maxInerval || delay == last {
			break
		}

		last = delay
	}

	assert.True(t, i < 100)
}

func TestUnmarshalClientSettings(t *testing.T) {
	config := cfg.New()

	err := config.Option(cfg.WithConfigMap(map[string]any{
		"cloud.aws": map[string]any{
			"defaults": map[string]any{
				"http_client.timeout": "1s",
				"assume_role":         "role",
				"credentials": map[string]any{
					"access_key_id":     "access key id",
					"secret_access_key": "secret access key",
					"session_token":     "session token",
				},
			},
			"cloudwatch.clients.metrics.http_client": map[string]any{
				"timeout": "2s",
			},
		},
	}))
	assert.NoError(t, err)

	settings := &aws.ClientSettings{}
	err = aws.UnmarshalClientSettings(config, settings, "cloudwatch", "default")
	assert.NoError(t, err)

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
	err = aws.UnmarshalClientSettings(config, settings, "cloudwatch", "metrics")
	assert.NoError(t, err)
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
