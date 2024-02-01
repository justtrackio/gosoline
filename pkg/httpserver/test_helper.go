package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

func HttpTest(method string, path string, requestPath string, body string, handler gin.HandlerFunc, requestOptions ...func(r *http.Request)) *httptest.ResponseRecorder {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(location.Default())
	r.Handle(method, path, handler)

	bodyReader := strings.NewReader(body)
	request, _ := http.NewRequest(method, requestPath, bodyReader)
	for _, opt := range requestOptions {
		opt(request)
	}

	response := httptest.NewRecorder()

	r.ServeHTTP(response, request)

	return response
}
