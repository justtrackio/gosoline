package http_test

import (
	"context"
	"fmt"
	netHttp "net/http"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/http"
	httpMocks "github.com/justtrackio/gosoline/pkg/http/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type circuitBreakerTestSuite struct {
	suite.Suite
	method string

	ctx        context.Context
	logger     logMocks.LoggerMock
	clock      clock.FakeClock
	baseClient *httpMocks.Client

	client http.Client
}

type outcome int

const (
	outcomeSuccess outcome = iota
	outcomeBadResponse
	outcomeConnectionError
	outcomeCanceled
	outcomeLimited
)

func TestCircuitBreakerClient(t *testing.T) {
	for _, method := range []string{"Get", "Post", "Put", "Patch", "Delete"} {
		suite.Run(t, &circuitBreakerTestSuite{
			method: method,
		})
	}
}

func (s *circuitBreakerTestSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.logger = logMocks.NewLoggerMock(logMocks.WithMockUntilLevel(log.PriorityWarn), logMocks.WithTestingT(s.T()))
	s.clock = clock.NewFakeClock()
	s.baseClient = httpMocks.NewClient(s.T())
	s.client = http.NewCircuitBreakerClientWithInterfaces(s.baseClient, s.logger, s.clock, "test", http.CircuitBreakerSettings{
		MaxFailures:      2,
		RetryDelay:       time.Minute,
		ExpectedStatuses: []int{netHttp.StatusOK},
	})
}

func (s *circuitBreakerTestSuite) TestSuccess() {
	s.performRequest(outcomeSuccess)
}

func (s *circuitBreakerTestSuite) TestSingleBadResponse() {
	s.performRequest(outcomeBadResponse)
}

func (s *circuitBreakerTestSuite) TestManyCanceledContext() {
	for i := 0; i < 10; i++ {
		s.performRequest(outcomeCanceled)
	}
}

func (s *circuitBreakerTestSuite) TestTriggerCircuitBreaker() {
	// perform some failing requests
	s.performRequest(outcomeBadResponse)
	s.performRequest(outcomeConnectionError)

	// breaker triggered
	for i := 0; i < 10; i++ {
		s.performRequest(outcomeLimited)
	}

	// advance time to try again
	s.clock.Advance(time.Minute)

	// still failing, but one request comes through
	s.performRequest(outcomeBadResponse)

	// all other requests are still rejected
	for i := 0; i < 10; i++ {
		s.performRequest(outcomeLimited)
	}

	// advance time again
	s.clock.Advance(time.Minute)

	// the first request opens the circuit again, now all requests succeed again
	for i := 0; i < 10; i++ {
		s.performRequest(outcomeSuccess)
	}
}

func (s *circuitBreakerTestSuite) performRequest(outcome outcome) {
	req := http.NewRequest(nil).WithUrl("http://localhost/" + uuid.New().NewV4())

	if outcome != outcomeLimited {
		var response *http.Response
		var err error

		switch outcome {
		case outcomeSuccess:
			response = &http.Response{
				StatusCode: netHttp.StatusOK,
			}
		case outcomeBadResponse:
			response = &http.Response{
				StatusCode: netHttp.StatusBadRequest,
			}
		case outcomeConnectionError:
			err = fmt.Errorf("conn error")
		case outcomeCanceled:
			err = context.Canceled
		}

		s.baseClient.On(s.method, s.ctx, req).Return(response, err).Once()
	}

	var res *http.Response
	var err error
	switch s.method {
	case "Get":
		res, err = s.client.Get(s.ctx, req)
	case "Post":
		res, err = s.client.Post(s.ctx, req)
	case "Put":
		res, err = s.client.Put(s.ctx, req)
	case "Patch":
		res, err = s.client.Patch(s.ctx, req)
	case "Delete":
		res, err = s.client.Delete(s.ctx, req)
	default:
		s.Fail("unknown method")
	}

	switch outcome {
	case outcomeSuccess:
		s.NoError(err)
		s.Equal(netHttp.StatusOK, res.StatusCode)
	case outcomeBadResponse:
		s.NoError(err)
		s.Equal(netHttp.StatusBadRequest, res.StatusCode)
	case outcomeConnectionError:
		s.EqualError(err, "conn error")
		s.Nil(res)
	case outcomeCanceled:
		s.EqualError(err, context.Canceled.Error())
		s.Nil(res)
	case outcomeLimited:
		s.EqualError(err, http.CircuitIsOpenError{}.Error())
		s.Nil(res)
	default:
		s.Fail("unknown outcome")
	}
}
