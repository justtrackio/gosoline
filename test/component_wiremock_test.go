//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func Test_wiremock(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.wiremock.test.yml")
	defer pkgTest.Shutdown()

	resp, err := http.Get("http://172.17.0.1:12345/__admin")

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
