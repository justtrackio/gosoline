//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func Test_elasticsearch(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.elasticsearch.test.yml")
	defer mocks.Shutdown()

	resp, err := http.Get("http://172.17.0.1:9201")

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
