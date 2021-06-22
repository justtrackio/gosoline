// +build integration

package apiserver

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/test/suite"
	"github.com/go-playground/validator/v10"
	"github.com/go-resty/resty/v2"
	netHttp "net/http"
	"reflect"
	"strconv"
	"testing"
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

type messageHandler struct{}
type customMessageHandler struct{}
type aliasMessageHandler struct{}
type nestedHandler struct{}
type structHandler struct{}

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

func (s *ValidationTestSuite) SetupApiDefinitions() apiserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		err := apiserver.AddCustomValidators(customValidators)
		if err != nil {
			return nil, err
		}

		err = apiserver.AddStructValidators(structValidators)
		if err != nil {
			return nil, err
		}

		err = apiserver.AddCustomTypeFuncs(customTypeFuncs)
		if err != nil {
			return nil, err
		}

		err = apiserver.AddValidateAlias(validateAliases)
		if err != nil {
			return nil, err
		}

		d := &apiserver.Definitions{}
		d.POST("/message", apiserver.CreateJsonHandler(&messageHandler{}))
		d.POST("/custom", apiserver.CreateJsonHandler(&customMessageHandler{}))
		d.POST("/alias", apiserver.CreateJsonHandler(&aliasMessageHandler{}))
		d.POST("/nested", apiserver.CreateJsonHandler(&nestedHandler{}))
		d.POST("/struct", apiserver.CreateJsonHandler(&structHandler{}))

		return d, nil
	}
}

func (s *ValidationTestSuite) TestValidateMessageInput_Success() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/message",
		Body: &ValidateMessageInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"1234"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateMessageInput_Failure() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/message",
		Body: &ValidateMessageInput{
			Message: "123",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateMessageInput.Message' Error:Field validation for 'Message' failed on the 'hasEvenLength' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateCustomInput_Success() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/custom",
		Body: &ValidateCustomInput{
			Message: "12345",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"12345"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateCustomInput_Failure() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/custom",
		Body: &ValidateCustomInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateCustomInput.Message' Error:Field validation for 'Message' failed on the 'len' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateAliasInput_Success() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/alias",
		Body: &ValidateAliasInput{
			Message: "12345",
		},
		ExpectedStatusCode: netHttp.StatusOK,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"message":"12345"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateAliasInput_Failure() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/alias",
		Body: &ValidateAliasInput{
			Message: "1234",
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateAliasInput.Message' Error:Field validation for 'Message' failed on the 'validMessage' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateNestedInput_Success() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/nested",
		Body: &ValidateNestedInput{
			Nested: ValidateInputNested{
				A: 13,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusOK,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"nested":{"a":13,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateNestedInput_Failure() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
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
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"nested":{"a":12,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateStructInput_Success() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/struct",
		Body: &ValidateStructInput{
			Struct: ValidateInputStruct{
				A: 13,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusOK,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"struct":{"a":13,"b":7}}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (s *ValidationTestSuite) TestValidateStructInput_Failure() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method: http.PostRequest,
		Url:    "/struct",
		Body: &ValidateStructInput{
			Struct: ValidateInputStruct{
				A: 12,
				B: 7,
			},
		},
		ExpectedStatusCode: netHttp.StatusBadRequest,
		ValidateResponse: func(response *resty.Response) error {
			// language=JSON
			expectedResult := `{"err":"Key: 'ValidateStructInput.Struct.A' Error:Field validation for 'A' failed on the 'hasEvenSum' tag\nKey: 'ValidateStructInput.Struct.B' Error:Field validation for 'B' failed on the 'hasEvenSum' tag"}`

			s.Equal(expectedResult, string(response.Body()))

			return nil
		},
	}
}

func (h messageHandler) GetInput() interface{} {
	return &ValidateMessageInput{}
}

func (h messageHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*ValidateMessageInput)

	return apiserver.NewJsonResponse(*body), nil
}

func (h customMessageHandler) GetInput() interface{} {
	return &ValidateCustomInput{}
}

func (h customMessageHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*ValidateCustomInput)

	return apiserver.NewJsonResponse(*body), nil
}

func (h aliasMessageHandler) GetInput() interface{} {
	return &ValidateAliasInput{}
}

func (h aliasMessageHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*ValidateAliasInput)

	return apiserver.NewJsonResponse(*body), nil
}

func (h nestedHandler) GetInput() interface{} {
	return &ValidateNestedInput{}
}

func (h nestedHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*ValidateNestedInput)

	return apiserver.NewJsonResponse(*body), nil
}

func (h structHandler) GetInput() interface{} {
	return &ValidateStructInput{}
}

func (h structHandler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*ValidateStructInput)

	return apiserver.NewJsonResponse(*body), nil
}

var customValidators = []apiserver.CustomValidator{
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

var structValidators = []apiserver.StructValidator{
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

var customTypeFuncs = []apiserver.CustomTypeFunc{
	{
		Func:  convertMessageType,
		Types: []interface{}{MessageType("")},
	},
}

func convertMessageType(field reflect.Value) interface{} {
	msgType := field.Interface().(MessageType)

	return string(msgType)
}

var validateAliases = []apiserver.ValidateAlias{
	{
		Alias: "validMessage",
		Tags:  "required,len=5",
	},
}

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, &ValidationTestSuite{})
}
