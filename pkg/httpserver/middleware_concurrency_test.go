package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConcurrentRequestLimitRouter(settings httpserver.ConcurrencySettings, entered chan struct{}, release <-chan struct{}) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var enteredOnce sync.Once

	router.Use(httpserver.ConcurrentRequestLimitMiddleware(settings))
	router.GET("/", func(c *gin.Context) {
		if entered != nil {
			enteredOnce.Do(func() {
				close(entered)
			})
		}

		if release != nil {
			<-release
		}

		c.Status(http.StatusNoContent)
	})

	return router
}

func TestConcurrentRequestLimitMiddleware_ZeroLimitDisablesEnforcement(t *testing.T) {
	release := make(chan struct{})
	router := newConcurrentRequestLimitRouter(httpserver.ConcurrencySettings{}, nil, release)

	result := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			recorder := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
			require.NoError(t, err)

			router.ServeHTTP(recorder, req)
			result <- recorder.Code
		}()
	}

	close(release)

	assert.Equal(t, http.StatusNoContent, <-result)
	assert.Equal(t, http.StatusNoContent, <-result)
}

func TestConcurrentRequestLimitMiddleware_RejectsWhenLimitReached(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	router := newConcurrentRequestLimitRouter(httpserver.ConcurrencySettings{
		MaxRequests:        1,
		OverloadStatusCode: http.StatusTooManyRequests,
		RetryAfter:         1500 * time.Millisecond,
	}, entered, release)

	firstResult := make(chan int)
	go func() {
		recorder := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)
		firstResult <- recorder.Code
	}()
	<-entered

	recorder := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
	require.NoError(t, err)

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Equal(t, "2", recorder.Header().Get("Retry-After"))
	assert.JSONEq(t, `{"error":"server overloaded"}`, recorder.Body.String())

	close(release)
	assert.Equal(t, http.StatusNoContent, <-firstResult)
}

func TestConcurrentRequestLimitMiddleware_ReleasesSlotAfterRequest(t *testing.T) {
	router := newConcurrentRequestLimitRouter(httpserver.ConcurrencySettings{MaxRequests: 1}, nil, nil)

	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/", http.NoBody)
		require.NoError(t, err)

		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusNoContent, recorder.Code)
	}
}
