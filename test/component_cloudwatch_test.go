//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_cloudwatch(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.cloudwatch.test.yml")
	defer mocks.Shutdown()

	cwClient := mocks.ProvideClient("cloudwatch", "cloudwatch").(*cloudwatch.CloudWatch)
	o, err := cwClient.ListDashboards(&cloudwatch.ListDashboardsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.DashboardEntries, 0)
}
