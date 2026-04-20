package httpserver_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
)

func makeParamRequest(paramName, paramValue string) *httpserver.Request {
	return &httpserver.Request{
		Params: gin.Params{{Key: paramName, Value: paramValue}},
	}
}

func TestGetUintFromRequest_RejectsNegativeValue(t *testing.T) {
	req := makeParamRequest("id", "-1")
	val, ok := httpserver.GetUintFromRequest(req, "id")
	assert.False(t, ok, "negative value must be rejected")
	assert.Equal(t, uint(0), *val)
}

func TestGetUintFromRequest_AcceptsZero(t *testing.T) {
	req := makeParamRequest("id", "0")
	val, ok := httpserver.GetUintFromRequest(req, "id")
	assert.True(t, ok)
	assert.Equal(t, uint(0), *val)
}

func TestGetUintFromRequest_AcceptsPositive(t *testing.T) {
	req := makeParamRequest("id", "42")
	val, ok := httpserver.GetUintFromRequest(req, "id")
	assert.True(t, ok)
	assert.Equal(t, uint(42), *val)
}

func TestGetUintFromRequest_RejectsNonNumeric(t *testing.T) {
	req := makeParamRequest("id", "abc")
	val, ok := httpserver.GetUintFromRequest(req, "id")
	assert.False(t, ok)
	assert.Equal(t, uint(0), *val)
}

func TestGetUintFromRequest_RejectsMissing(t *testing.T) {
	req := makeParamRequest("other", "1")
	val, ok := httpserver.GetUintFromRequest(req, "id")
	assert.False(t, ok)
	assert.Equal(t, uint(0), *val)
}
