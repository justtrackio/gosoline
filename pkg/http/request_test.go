package http_test

import (
	"net/url"
	"testing"

	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/stretchr/testify/assert"
)

func TestRequest_WithAll(t *testing.T) {
	request := http.NewRequest(nil).
		WithUrl("example.com/foo?some=key").
		WithQueryParam("key", "value").
		WithQueryMap(map[string]string{
			"my-key": "my-value",
		}).
		WithBody("{}").
		WithAuthToken("token").
		WithHeader("X-API-KEY", "api-key").
		WithHeader("X-API-VERSION", "42")

	err := request.GetError()
	assert.NoError(t, err)

	assert.Equal(t, "token", request.GetToken())
	assert.Equal(t, http.Header{
		"X-Api-Key":     {"api-key"},
		"X-Api-Version": {"42"},
	}, request.GetHeader())
	assert.Equal(t, "example.com/foo?key=value&my-key=my-value&some=key", request.GetUrl())
	assert.Equal(t, "{}", request.GetBody())
}

type TestQueryParams struct {
	Name       string `url:"name"`
	RangeStart int    `url:"range"`
	RangeEnd   int    `url:"range"`
	User       struct {
		Age    int    `url:"age"`
		Gender string `url:"gender"`
	} `url:"user"`
	Aliases []string `url:"aliases[]"`
	Friends []struct {
		Name string `url:"name"`
	} `url:"friends"`
}

func TestRequest_WithQueryObject(t *testing.T) {
	data := TestQueryParams{
		Name:       "test",
		RangeStart: 5,
		RangeEnd:   10,
		User: struct {
			Age    int    `url:"age"`
			Gender string `url:"gender"`
		}{
			Age:    23,
			Gender: "m",
		},
		Aliases: []string{
			"test-user",
			"tester",
		},
		Friends: []struct {
			Name string `url:"name"`
		}{
			{
				Name: "qa",
			},
			{
				Name: "ci",
			},
		},
	}

	request := http.NewRequest(nil).
		WithQueryObject(data)

	err := request.GetError()
	assert.NoError(t, err)

	parsedUrl, err := url.Parse(request.GetUrl())
	assert.NoError(t, err)

	assert.Equal(t, url.Values{
		"name": []string{"test"},
		"range": []string{
			"5",
			"10",
		},
		"user[age]": []string{
			"23",
		},
		"user[gender]": []string{
			"m",
		},
		"aliases[]": []string{
			"test-user",
			"tester",
		},
		"friends": []string{
			"{qa}",
			"{ci}",
		},
	}, parsedUrl.Query())
}

func TestRequest_GetUrl(t *testing.T) {
	request := http.NewRequest(nil).
		WithUrl("https://justtrack.io?test999=1").
		WithQueryParam("test", "test1", "test2").
		WithQueryParam("test2", 1, 2.2, "test")

	err := request.GetError()
	assert.NoError(t, err)

	expected := "https://justtrack.io?test=test1&test=test2&test2=1&test2=2.2&test2=test&test999=1"
	assert.Equal(t, expected, request.GetUrl())
}

func TestRequest_HandleQueryParamsCorrectly(t *testing.T) {
	// just creating a request for a URL directly doesn't cause : to get encoded

	request := http.NewRequest(nil).
		WithUrl("https://example.com?api_key=foo:bar")
	assert.Equal(t, "https://example.com?api_key=foo:bar", request.GetUrl())

	// now we add a query parameter in different ways, it always causes the : to get encoded

	// WithQueryParam

	request = http.NewRequest(nil).
		WithUrl("https://example.com?api_key=foo:bar").
		WithQueryParam("data", "42")
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())

	request = http.NewRequest(nil).
		WithQueryParam("data", "42").
		WithUrl("https://example.com?api_key=foo:bar")
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())

	// WithQueryMap

	request = http.NewRequest(nil).
		WithUrl("https://example.com?api_key=foo:bar").
		WithQueryMap(map[string]any{
			"data": "42",
		})
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())

	request = http.NewRequest(nil).
		WithQueryMap(map[string]any{
			"data": "42",
		}).
		WithUrl("https://example.com?api_key=foo:bar")
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())

	// WithQueryObject

	type QueryObject struct {
		Data string `url:"data"`
	}

	request = http.NewRequest(nil).
		WithUrl("https://example.com?api_key=foo:bar").
		WithQueryObject(QueryObject{
			Data: "42",
		})
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())

	request = http.NewRequest(nil).
		WithQueryObject(QueryObject{
			Data: "42",
		}).
		WithUrl("https://example.com?api_key=foo:bar")
	assert.Equal(t, "https://example.com?api_key=foo%3Abar&data=42", request.GetUrl())
}
