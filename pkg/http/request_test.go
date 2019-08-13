package http_test

import (
	"github.com/applike/gosoline/pkg/http"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestRequest_WithAll(t *testing.T) {
	request := http.NewRequest().
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

	request := http.NewRequest().
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
	request := http.NewRequest().
		WithUrl("https://applike.info?test999=1").
		WithQueryParam("test", "test1", "test2").
		WithQueryParam("test2", 1, 2.2, "test")

	err := request.GetError()
	assert.NoError(t, err)

	expected := "https://applike.info?test=test1&test=test2&test2=1&test2=2.2&test2=test&test999=1"
	assert.Equal(t, expected, request.GetUrl())
}
