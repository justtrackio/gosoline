// +build integration

package apiserver

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type handler struct{}

type Input struct {
	Message string `json:"message" binding:"hasEvenLength"`
}

type testModule struct {
	kernel.EssentialModule

	t      *testing.T
	client http.Client
}

func TestValidation(t *testing.T) {
	args := os.Args
	os.Args = args[:1]
	defer func() {
		os.Args = args
	}()

	app := application.Default()
	app.Add("api", apiserver.New(defineRoutes))
	app.Add("test", newTestModule(t))
	app.Run()
}

func defineRoutes(_ context.Context, _ cfg.Config, _ mon.Logger) (*apiserver.Definitions, error) {
	err := apiserver.AddCustomValidators(customValidators)
	if err != nil {
		return nil, err
	}

	d := &apiserver.Definitions{}
	d.POST("/echo", apiserver.CreateJsonHandler(&handler{}))

	return d, nil
}

var customValidators = []apiserver.CustomValidator{
	{
		Name:      "hasEvenLength",
		Validator: hasEvenLengthValidator,
	},
}

func hasEvenLengthValidator(fl validator.FieldLevel) bool {
	input, ok := fl.Field().Interface().(string)

	if !ok {
		return false
	}

	return len(input)%2 == 0
}

func (h handler) GetInput() interface{} {
	return &Input{}
}

func (h handler) Handle(_ context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	body := request.Body.(*Input)

	return apiserver.NewJsonResponse(*body), nil
}

func newTestModule(t *testing.T) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		client := http.NewHttpClient(config, logger)

		return &testModule{
			t:      t,
			client: client,
		}, nil
	}
}

func (t testModule) Run(ctx context.Context) error {
	// language=JSON
	bodyString := `{ "message": "1234" }`
	// language=JSON
	expectedBody := `{"message":"1234"}`

	req := http.NewRequest(t.client).
		WithUrl("http://localhost:25398/echo").
		WithBody([]byte(bodyString)).
		WithHeader(http.HdrContentType, http.ContentTypeApplicationJson)

	res, err := t.client.Post(ctx, req)
	assert.NoError(t.t, err)

	assert.Equal(t.t, apiserver.ContentTypeJson, res.Header.Get(http.HdrContentType))
	assert.Equal(t.t, expectedBody, string(res.Body))

	// language=JSON
	bodyString = `{ "message": "123" }`
	// language=JSON
	expectedBody = `{"err":"Key: 'Input.Message' Error:Field validation for 'Message' failed on the 'hasEvenLength' tag"}`

	req = http.NewRequest(t.client).
		WithUrl("http://localhost:25398/echo").
		WithBody([]byte(bodyString)).
		WithHeader(http.HdrContentType, http.ContentTypeApplicationJson)

	res, err = t.client.Post(ctx, req)
	assert.NoError(t.t, err)

	assert.Equal(t.t, apiserver.ContentTypeJson, res.Header.Get(http.HdrContentType))
	assert.Equal(t.t, expectedBody, string(res.Body))

	return nil
}
