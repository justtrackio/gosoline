//go:build integration
// +build integration

package test_test

import (
	"fmt"
	"net/http"
	"testing"

	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
)

func Test_wiremock(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.wiremock.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	host := mocks.ProvideWiremockHost("wiremock")
	port := mocks.ProvideWiremockPort("wiremock")
	url := fmt.Sprintf("http://%s:%d%s", host, port, "/__admin")
	resp, err := http.Get(url)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
