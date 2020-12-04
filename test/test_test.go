//+build integration

package test_test

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func setup(t *testing.T) {
	err := os.Setenv("AWS_ACCESS_KEY_ID", "gosoline")
	assert.NoError(t, err)

	err = os.Setenv("AWS_SECRET_ACCESS_KEY", "gosoline")
	assert.NoError(t, err)
}
