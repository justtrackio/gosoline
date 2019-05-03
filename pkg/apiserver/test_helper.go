package apiserver

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
)

func HttpTest(method string, path string, requestPath string, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Handle(method, path, handler)

	bodyReader := strings.NewReader(body)
	request, _ := http.NewRequest(method, requestPath, bodyReader)
	response := httptest.NewRecorder()

	r.ServeHTTP(response, request)

	return response
}
