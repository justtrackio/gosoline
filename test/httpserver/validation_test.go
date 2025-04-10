//go:build integration

package httpserver

import (
	"context"
	netHttp "net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ValidateMessageInput struct {
	Message string `json:"message" binding:"hasEvenLength"`
}

type MessageType string

type ValidateCustomInput struct {
	Message MessageType `json:"message" binding:"required,len=5"`
}

type ValidateAliasInput struct {
	Message string `json:"message" binding:"validMessage"`
}

type ValidateNestedInput struct {
	Nested ValidateInputNested `json:"nested" binding:"hasEvenSum"`
}

type ValidateStructInput struct {
	Struct ValidateInputStruct `json:"struct"`
}

type ValidateInputNested struct {
	A int `json:"a"`
	B int `json:"b"`
}

type ValidateInputStruct struct {
	A int `json:"a"`
	B int `json:"b"`
}

type (
	messageHandler       struct{}
	customMessageHandler struct{}
	aliasMessageHandler  struct{}
	nestedHandler        struct{}
	structHandler        struct{}
)

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, &ValidationTestSuite{})
}

type ValidationTestSuite struct {
	suite.Suite
}

func (s *ValidationTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("./config.dist.validation.yml"),
		suite.WithSharedEnvironment(),
		suite.WithoutAutoDetectedComponents("localstack"),
	}
}

func (s *ValidationTestSuite) SetupApiDefinitions() httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		err := httpserver.AddCustomValidators(customValidators)
		if err != nil {
			return nil, err
		}

		err = httpserver.AddStructValidators(structValidators)
		if err != nil {
			return nil, err
		}

		err = httpserver.AddCustomTypeFuncs(customTypeFuncs)
		if err != nil {
			return nil, err
		}

		err = httpserver.AddValidateAlias(validateAliases)
		if err != nil {
			return nil, err
		}

		d := &httpserver.Definitions{}
		d.POST("/message", httpserver.CreateJsonHandler(&messageHandler{}))
		d.POST("/custom", httpserver.CreateJsonHandler(&customMessageHandler{}))
		d.POST("/alias", httpserver.CreateJsonHandler(&aliasMessageHandler{}))
		d.POST("/nested", httpserver.CreateJsonHandler(&nestedHandler{}))
		d.POST("/struct", httpserver.CreateJsonHandler(&structHandler{}))

		return d, nil
	}
}

func (s *ValidationTestSuite) TestValidateMessageInput_Success() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/message",
		Body: &ValidateMessageInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"1234"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateMessageInput_Failure() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/message",
		Body: &ValidateMessageInput{
			Message: "123",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateMessageInput.Message' Error:Field validation for 'Message' failed on the 'hasEvenLength' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateCustomInput_Success() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/custom",
		Body: &ValidateCustomInput{
			Message: "12345",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"12345"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateCustomInput_Failure() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/custom",
		Body: &ValidateCustomInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateCustomInput.Message' Error:Field validation for 'Message' failed on the 'len' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateAliasInput_Success() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/alias",
		Body: &ValidateAliasInput{
			Message: "12345",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"12345"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateAliasInput_Failure() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/alias",
		Body: &ValidateAliasInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateAliasInput.Message' Error:Field validation for 'Message' failed on the 'validMessage' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateNestedInput_Success() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/nested",
		Body: &ValidateNestedInput{
			Nested: ValidateInputNested{
				A: 13,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"nested":{"a":13,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateNestedInput_Failure() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/nested",
		Body: &ValidateNestedInput{
			Nested: ValidateInputNested{
				A: 12,
				B: 7,
			},
		},
		// We can't yet validate nested structs, so this returns success for now. This test ensures we notice if this
		// state changes.
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"nested":{"a":12,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateStructInput_Success() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/struct",
		Body: &ValidateStructInput{
			Struct: ValidateInputStruct{
				A: 13,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"struct":{"a":13,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateStructInput_Failure() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method: http.PostRequest,
		Url:    "/struct",
		Body: &ValidateStructInput{
			Struct: ValidateInputStruct{
				A: 12,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateStructInput.Struct.A' Error:Field validation for 'A' failed on the 'hasEvenSum' tag\nKey: 'ValidateStructInput.Struct.B' Error:Field validation for 'B' failed on the 'hasEvenSum' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (h messageHandler) GetInput() any {
	return &ValidateMessageInput{}
}

func (h messageHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*ValidateMessageInput)

	return httpserver.NewJsonResponse(*body), nil
}

func (h customMessageHandler) GetInput() any {
	return &ValidateCustomInput{}
}

func (h customMessageHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*ValidateCustomInput)

	return httpserver.NewJsonResponse(*body), nil
}

func (h aliasMessageHandler) GetInput() any {
	return &ValidateAliasInput{}
}

func (h aliasMessageHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*ValidateAliasInput)

	return httpserver.NewJsonResponse(*body), nil
}

func (h nestedHandler) GetInput() any {
	return &ValidateNestedInput{}
}

func (h nestedHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*ValidateNestedInput)

	return httpserver.NewJsonResponse(*body), nil
}

func (h structHandler) GetInput() any {
	return &ValidateStructInput{}
}

func (h structHandler) Handle(_ context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	body := request.Body.(*ValidateStructInput)

	return httpserver.NewJsonResponse(*body), nil
}

var customValidators = []httpserver.CustomValidator{
	{
		Name:      "hasEvenLength",
		Validator: hasEvenLengthValidator,
	},
	{
		Name:      "hasEvenSum",
		Validator: hasEvenSumValidator,
	},
}

func hasEvenLengthValidator(fl validator.FieldLevel) bool {
	input, ok := fl.Field().Interface().(string)

	if !ok {
		return false
	}

	return len(input)%2 == 0
}

func hasEvenSumValidator(fl validator.FieldLevel) bool {
	input, ok := fl.Field().Interface().(ValidateInputNested)

	if !ok {
		return false
	}

	return (input.A+input.B)%2 == 0
}

var structValidators = []httpserver.StructValidator{
	{
		Struct:    &ValidateInputStruct{},
		Validator: hasEventSumStructValidator,
	},
}

func hasEventSumStructValidator(sl validator.StructLevel) {
	input := sl.Current().Interface().(ValidateInputStruct)

	if (input.A+input.B)%2 != 0 {
		sl.ReportError(input.A, "A", "A", "hasEvenSum", strconv.Itoa(input.A))
		sl.ReportError(input.B, "B", "B", "hasEvenSum", strconv.Itoa(input.B))
	}
}

var customTypeFuncs = []httpserver.CustomTypeFunc{
	{
		Func:  convertMessageType,
		Types: []any{MessageType("")},
	},
}

func convertMessageType(field reflect.Value) any {
	msgType := field.Interface().(MessageType)

	return string(msgType)
}

var validateAliases = []httpserver.ValidateAlias{
	{
		Alias: "validMessage",
		Tags:  "required,len=5",
	},
}
