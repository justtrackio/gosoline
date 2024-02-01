package httpserver_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
)

type Input struct {
	Text string `json:"text" binding:"required"`
}

type Output struct {
	Text string `json:"text"`
}

type HtmlHandler struct{}

func (h HtmlHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	out := fmt.Sprintf("<html><body>%s</body></html>", request.Body)

	return httpserver.NewHtmlResponse(out), nil
}

type JsonHandler struct{}

func (h JsonHandler) GetInput() interface{} {
	return &Input{}
}

func (h JsonHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	inp := request.Body.(*Input)
	out := Output{
		Text: inp.Text,
	}

	return httpserver.NewJsonResponse(out), nil
}

type RedirectHandler struct{}

func (h RedirectHandler) Handle(_ context.Context, _ *httpserver.Request) (*httpserver.Response, error) {
	return httpserver.NewRedirectResponse("https://example.com"), nil
}

type NotModifiedHandler struct{}

func (h NotModifiedHandler) Handle(_ context.Context, _ *httpserver.Request) (*httpserver.Response, error) {
	return httpserver.NewStatusResponse(http.StatusNotModified), nil
}

func TestHtmlHandler(t *testing.T) {
	handler := httpserver.CreateRawHandler(HtmlHandler{})
	response := httpserver.HttpTest("PUT", "/action", "/action", `foobar`, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, httpserver.ContentTypeHtml, response.Header().Get("Content-Type"))
	assert.Equal(t, `<html><body>foobar</body></html>`, response.Body.String())
}

func TestCreateIoHandler_InputFailure(t *testing.T) {
	handler := httpserver.CreateJsonHandler(JsonHandler{})
	response := httpserver.HttpTest("PUT", "/action", "/action", `{}`, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, `{"err":"Key: 'Input.Text' Error:Field validation for 'Text' failed on the 'required' tag"}`, response.Body.String())
}

func TestCreateIoHandler(t *testing.T) {
	handler := httpserver.CreateJsonHandler(JsonHandler{})
	response := httpserver.HttpTest("PUT", "/action", "/action", `{"text":"foobar"}`, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, `{"text":"foobar"}`, response.Body.String())
}

func TestRedirectHandler(t *testing.T) {
	handler := httpserver.CreateRawHandler(RedirectHandler{})
	response := httpserver.HttpTest("GET", "/redirect", "/redirect", "", handler)

	assert.Equal(t, http.StatusFound, response.Code)
	assert.Equal(t, "", response.Header().Get("Content-Type"))
	assert.Equal(t, "https://example.com", response.Header().Get("Location"))
	assert.Equal(t, "", response.Body.String())
}

func TestNotModifiedHandler(t *testing.T) {
	handler := httpserver.CreateRawHandler(NotModifiedHandler{})
	response := httpserver.HttpTest("GET", "/", "/", "", handler)

	assert.Equal(t, http.StatusNotModified, response.Code)
	assert.Equal(t, "", response.Header().Get("Content-Type"))
	assert.Equal(t, "", response.Header().Get("Location"))
	assert.Equal(t, "", response.Body.String())
}

type CookieHandler struct{}

func (h CookieHandler) Handle(_ context.Context, req *httpserver.Request) (*httpserver.Response, error) {
	return httpserver.NewJsonResponse(req.Cookies), nil
}

func TestCookieHandler(t *testing.T) {
	handler := httpserver.CreateRawHandler(CookieHandler{})
	response := httpserver.HttpTest("GET", "/", "/", "", handler, func(r *http.Request) {
		r.Header.Set("Cookie", "cookie1=value1;cookie2=value2;cookie1=value3")
	})

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, httpserver.ContentTypeJson, response.Header().Get("Content-Type"))
	assert.Equal(t, "", response.Header().Get("Location"))
	assert.JSONEq(t, `{"cookie1":"value3","cookie2":"value2"}`, response.Body.String())
}
