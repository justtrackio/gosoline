//+build integration

package test_test

import (
	"fmt"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func Test_wiremock(t *testing.T) {
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.wiremock.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks")

		return
	}

	port := mocks.ProvideWiremockPort("wiremock")
	url := fmt.Sprintf("http://%s:%d%s", "172.17.0.1", port, "/__admin")
	resp, err := http.Get(url)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
