//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_cloudwatch(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.cloudwatch.test.yml"))

	cwClient := test.ProvideCloudwatchClient("cloudwatch")
	o, err := cwClient.ListDashboards(&cloudwatch.ListDashboardsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.DashboardEntries, 0)

	test.Shutdown()
}
