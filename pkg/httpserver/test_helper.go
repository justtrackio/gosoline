package httpserver

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

//go:generate go run github.com/vektra/mockery/v2 --name NetListener
type NetListener interface {
	net.Listener
}

//go:generate go run github.com/vektra/mockery/v2 --name NetConn
type NetConn interface {
	net.Conn
}

var (
	_ NetListener = nil
	_ NetConn     = nil
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

	request, err := http.NewRequest(method, requestPath, bodyReader)
	if err != nil {
		panic(fmt.Sprintf("httpserver.HttpTest: failed to create request: %v", err))
	}

	for _, opt := range requestOptions {
		opt(request)
	}

	response := httptest.NewRecorder()

	r.ServeHTTP(response, request)

	return response
}
