//go:build integration
// +build integration

package test_test

import (
	"os"
	"testing"

	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	err := os.Setenv("AWS_ACCESS_KEY_ID", gosoAws.DefaultAccessKeyID)
	assert.NoError(t, err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", gosoAws.DefaultSecretAccessKey)
	assert.NoError(t, err)
}
