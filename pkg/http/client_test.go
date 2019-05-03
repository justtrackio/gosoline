package http_test

import (
	"github.com/applike/gosoline/pkg/http"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRequest_GetUrl(t *testing.T) {
	test := []interface{}{
		"test1",
		"test2",
	}

	test2 := []interface{}{
		1,
		2.2,
		"test",
	}

	request := http.NewRequest("https://applike.info?test999=1")
	request.QueryParams = http.QueryParams{
		"test":  test,
		"test2": test2,
	}

	url, err := request.GetUrl()

	expected := "https://applike.info?test=test1&test=test2&test2=1&test2=2.2&test2=test&test999=1"
	assert.Equal(t, expected, url)

	assert.NoError(t, err)
}
