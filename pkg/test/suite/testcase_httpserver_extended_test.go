package suite_test

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/pkg/test/suite/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type GatewayTestSuite struct {
	suite.Suite
	totalTests int32
}

//go:generate protoc --go_out=.. testcase_httpserver_extended_test_input.proto
type TestInput struct {
	Text string `json:"text" binding:"required"`
}

func (p *TestInput) ToMessage() (proto.Message, error) {
	return &testdata.TestInput{
		Text: p.Text,
	}, nil
}

func (p *TestInput) EmptyMessage() proto.Message {
	return &testdata.TestInput{}
}

func (p *TestInput) FromMessage(message proto.Message) error {
	input := message.(*testdata.TestInput)
	p.Text = input.GetText()

	return nil
}

func TestGatewayTestSuite(t *testing.T) {
	var s GatewayTestSuite
	suite.Run(t, &s)
	assert.Equal(t, int32(16), s.totalTests)
}

func (s *GatewayTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithSharedEnvironment(),
	}
}

func (s *GatewayTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}

		d.GET("/noop", func(ginCtx *gin.Context) {
			ginCtx.String(http.StatusOK, "{}")
		})

		d.POST("/echo", func(ginCtx *gin.Context) {
			body, err := io.ReadAll(ginCtx.Request.Body)
			s.NoError(err)

			contentType := ginCtx.ContentType()

			ginCtx.Data(http.StatusOK, contentType, body)
		})

		d.POST("/reverse", func(ginCtx *gin.Context) {
			body, err := io.ReadAll(ginCtx.Request.Body)
			s.NoError(err)

			contentType := ginCtx.ContentType()

			ginCtx.Data(http.StatusOK, contentType, funk.Reverse(body))
		})

		return d, nil
	}
}

func (s *GatewayTestSuite) TestCaseMap() map[string]*suite.HttpserverTestCase {
	return map[string]*suite.HttpserverTestCase{
		"AnotherSingleTest": s.createTestCase(),
		"EmptyTest":         nil,
	}
}

func (s *GatewayTestSuite) TestCasesMap() map[string][]*suite.HttpserverTestCase {
	return map[string][]*suite.HttpserverTestCase{
		"AnotherSingleTest": {
			s.createTestCase(),
		},
		"AnotherMultipleTests": {
			s.createTestCase(),
			s.createProtobufTestCase(),
			s.createFileTestCase(),
		},
		"EmptyTest": {},
		"AnotherSkippedTest": {
			nil,
		},
	}
}

func (s *GatewayTestSuite) TestCasesMapWithProvider() map[string]suite.ToHttpserverTestCaseList {
	return map[string]suite.ToHttpserverTestCaseList{
		"AnotherSingleTest": s.createTestCase(),
		"AnotherMultipleTests": suite.HttpserverTestCaseListProvider(func() []*suite.HttpserverTestCase {
			return []*suite.HttpserverTestCase{
				s.createTestCase(),
				s.createProtobufTestCase(),
				s.createFileTestCase(),
			}
		}),
		"Nil": nil,
	}
}

func (s *GatewayTestSuite) TestCasesEmptyMap() map[string][]*suite.HttpserverTestCase {
	return map[string][]*suite.HttpserverTestCase{}
}

func (s *GatewayTestSuite) TestSingleTest() *suite.HttpserverTestCase {
	return s.createTestCase()
}

func (s *GatewayTestSuite) TestSkipped() *suite.HttpserverTestCase {
	return nil
}

func (s *GatewayTestSuite) TestNilProvider() suite.ToHttpserverTestCaseList {
	return nil
}

func (s *GatewayTestSuite) TestProvider() suite.ToHttpserverTestCaseList {
	return s.createTestCase()
}

func (s *GatewayTestSuite) TestMultipleTests() []*suite.HttpserverTestCase {
	return []*suite.HttpserverTestCase{
		s.createTestCase(),
		s.createProtobufTestCase(),
		s.createFileTestCase(),
	}
}

func (s *GatewayTestSuite) TestMultipleTestsWithNil() []*suite.HttpserverTestCase {
	return []*suite.HttpserverTestCase{
		s.createTestCase(),
		nil,
		s.createFileTestCase(),
	}
}

func (s *GatewayTestSuite) createTestCase() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method:             http.MethodGet,
		Url:                "/noop",
		Headers:            map[string]string{},
		Body:               struct{}{},
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResponse := `{}`
			s.JSONEq(expectedResponse, string(response.Body()))
			atomic.AddInt32(&s.totalTests, 1)

			return nil
		},
	}
}

func (s *GatewayTestSuite) createProtobufTestCase() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method:  http.MethodPost,
		Url:     "/echo",
		Headers: map[string]string{},
		Body: suite.EncodeBodyProtobuf(&TestInput{
			Text: "hello, world",
		}),
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			body := &TestInput{}
			msg := body.EmptyMessage()

			err := proto.Unmarshal(response.Body(), msg)
			s.NoError(err)

			err = body.FromMessage(msg)
			s.NoError(err)

			expectedResponse := &TestInput{
				Text: "hello, world",
			}
			s.Equal(expectedResponse, body)
			atomic.AddInt32(&s.totalTests, 1)

			return nil
		},
	}
}

func (s *GatewayTestSuite) createFileTestCase() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method:             http.MethodPost,
		Url:                "/echo",
		Headers:            map[string]string{},
		Body:               suite.ReadBodyFile("testdata/hello world.json"),
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResponse := `{"text": "hello, world"}`
			s.JSONEq(expectedResponse, string(response.Body()))
			atomic.AddInt32(&s.totalTests, 1)

			return nil
		},
	}
}
