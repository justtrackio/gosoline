package httpserver

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

type HttpBody interface {
	string | []byte
}

func HttpTest[Body HttpBody](method, path, requestPath string, body Body, handler gin.HandlerFunc, requestOptions ...func(r *http.Request)) *httptest.ResponseRecorder {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(location.Default())
	r.Handle(method, path, handler)

	var bodyReader io.Reader
	switch value := any(body).(type) {
	case string:
		bodyReader = strings.NewReader(value)
	case []byte:
		bodyReader = bytes.NewReader(value)
	}

	request, _ := http.NewRequest(method, requestPath, bodyReader)
	for _, opt := range requestOptions {
		opt(request)
	}

	response := httptest.NewRecorder()

	r.ServeHTTP(response, request)

	return response
}
