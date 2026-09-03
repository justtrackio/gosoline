package http

import (
	"context"
	stdHttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
)

func TestClientWritesRequestDurationForCanceledRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(stdHttp.HandlerFunc(func(_ stdHttp.ResponseWriter, request *stdHttp.Request) {
		close(requestStarted)
		<-request.Context().Done()
	}))
	t.Cleanup(server.Close)

	writer := cancellationMetricWriter{datums: make(chan *metric.Datum, 1)}
	client := NewHttpClientWithInterfaces(log.NewCliLogger(), clock.NewFakeClock(), writer, resty.New(), false, 0)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	result := make(chan httpClientResult, 1)
	go func() {
		response, err := client.Get(ctx, client.NewRequest().WithUrl(server.URL))
		result <- httpClientResult{response: response, err: err}
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("HTTP request did not reach the test server")
	}
	cancel()

	response := <-result
	assert.ErrorIs(t, response.err, context.Canceled)
	assert.Nil(t, response.response)

	datum := receiveCancellationMetric(t, writer.datums)
	assert.Equal(t, metricRequestDuration, datum.MetricName)
	assert.Equal(t, GetRequest, datum.Dimensions[dimensionMethod])
	assert.Equal(t, metric.DimensionDefault, datum.Dimensions[dimensionStatusCode])
	assert.Equal(t, metric.ErrorType(context.Canceled), datum.Dimensions[metric.DimensionErrorType])
}

func receiveCancellationMetric(t *testing.T, datums <-chan *metric.Datum) *metric.Datum {
	t.Helper()

	select {
	case datum := <-datums:
		return datum
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancellation request-duration metric")

		return nil
	}
}

type httpClientResult struct {
	response *Response
	err      error
}

type cancellationMetricWriter struct {
	datums chan *metric.Datum
}

func (w cancellationMetricWriter) GetPriority() int {
	return metric.PriorityLow
}

func (w cancellationMetricWriter) Write(ctx context.Context, data metric.Data) {
	for _, datum := range data {
		w.WriteOne(ctx, datum)
	}
}

func (w cancellationMetricWriter) WriteOne(_ context.Context, datum *metric.Datum) {
	w.datums <- datum
}
