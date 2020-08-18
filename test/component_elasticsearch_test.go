//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_elasticsearch(t *testing.T) {
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.elasticsearch.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	client := mocks.ProvideElasticsearchV6Client("elasticsearch", "default")
	resp, err := client.Info()

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
