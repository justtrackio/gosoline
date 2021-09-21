//go:build integration
// +build integration

package test_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
)

func Test_cloudwatch(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.cloudwatch.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	cwClient := mocks.ProvideCloudwatchClient("cloudwatch")
	o, err := cwClient.ListDashboards(&cloudwatch.ListDashboardsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.DashboardEntries, 0)
}
