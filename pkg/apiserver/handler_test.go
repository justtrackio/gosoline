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

func (h HtmlHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	out := fmt.Sprintf("<html><body>%s</body></html>", request.Body)

	return apiserver.NewHtmlResponse(out), nil
}

type JsonHandler struct {
}

func (h JsonHandler) GetInput() interface{} {
	return &Input{}
}

func (h JsonHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	inp := request.Body.(*Input)
	out := Output{
		Text: inp.Text,
	}

	return apiserver.NewJsonResponse(out), nil
}

type RedirectHandler struct {
}

func (h RedirectHandler) Handle(_ context.Context, _ *apiserver.Request) (*apiserver.Response, error) {
	return apiserver.NewRedirectResponse("https://example.com"), nil
}

type NotModifiedHandler struct {
}

func (h NotModifiedHandler) Handle(_ context.Context, _ *apiserver.Request) (*apiserver.Response, error) {
	return apiserver.NewStatusResponse(http.StatusNotModified), nil
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

func TestRedirectHandler(t *testing.T) {
	handler := apiserver.CreateRawHandler(RedirectHandler{})
	response := apiserver.HttpTest("GET", "/redirect", "/redirect", "", handler)

	assert.Equal(t, http.StatusFound, response.Code)
	assert.Equal(t, "", response.Header().Get("Content-Type"))
	assert.Equal(t, "https://example.com", response.Header().Get("Location"))
	assert.Equal(t, "", response.Body.String())
}

func TestNotModifiedHandler(t *testing.T) {
	handler := apiserver.CreateRawHandler(NotModifiedHandler{})
	response := apiserver.HttpTest("GET", "/", "/", "", handler)

	assert.Equal(t, http.StatusNotModified, response.Code)
	assert.Equal(t, "", response.Header().Get("Content-Type"))
	assert.Equal(t, "", response.Header().Get("Location"))
	assert.Equal(t, "", response.Body.String())
}
