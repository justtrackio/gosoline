package apiserver_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type Input struct {
	Text string `json:"text" binding:"required"`
}

type Output struct {
	Text string `json:"text"`
}

type HtmlHandler struct {
}

func (h HtmlHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	out := fmt.Sprintf("<html><body>%s</body></html>", request.Body)

	return apiserver.NewHtmlResponse(out), nil
}

type JsonHandler struct {
}

func (h JsonHandler) GetInput() interface{} {
	return &Input{}
}

func (h JsonHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	inp := request.Body.(*Input)
	out := Output{
		Text: inp.Text,
	}

	return apiserver.NewJsonResponse(out), nil
}

func TestHtmlHandler(t *testing.T) {
	handler := apiserver.CreateRawHandler(HtmlHandler{})
	response := apiserver.HttpTest("PUT", "/action", "/action", `foobar`, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, apiserver.ContentTypeHtml, response.Header().Get("Content-Type"))
	assert.Equal(t, `<html><body>foobar</body></html>`, response.Body.String())
}

func TestCreateIoHandler_InputFailure(t *testing.T) {
	handler := apiserver.CreateJsonHandler(JsonHandler{})
	response := apiserver.HttpTest("PUT", "/action", "/action", `{}`, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, `{"err":"Key: 'Input.Text' Error:Field validation for 'Text' failed on the 'required' tag"}`, response.Body.String())
}

func TestCreateIoHandler(t *testing.T) {
	handler := apiserver.CreateJsonHandler(JsonHandler{})
	response := apiserver.HttpTest("PUT", "/action", "/action", `{"text":"foobar"}`, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, `{"text":"foobar"}`, response.Body.String())
}
