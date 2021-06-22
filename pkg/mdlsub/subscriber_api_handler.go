package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"net/http"
)

type Handler struct {
	callback stream.ConsumerCallback
	tracer   tracing.Tracer
}

func NewHandler(ctx context.Context, config cfg.Config, logger log.Logger, callbackFactory stream.ConsumerCallbackFactory) (apiserver.HandlerWithInput, error) {
	callback, err := callbackFactory(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, err
	}

	return &Handler{
		callback: callback,
		tracer:   tracer,
	}, nil
}

func (h *Handler) GetInput() interface{} {
	return &stream.Message{}
}

func (h *Handler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	msg := request.Body.(*stream.Message)

	var model interface{}

	if model = h.callback.GetModel(msg.Attributes); model == nil {
		err := fmt.Errorf("invalid or incomplete attributes: %v", msg.Attributes)
		return apiserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("could not get model: %w", err)
	}

	encoding, err := stream.GetEncodingAttribute(msg.Attributes)
	if err != nil {
		return apiserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("could not get encoding: %w", err)
	}

	if encoding == nil {
		return apiserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("missing encoding attribute")
	}

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: *encoding,
	})

	var attributes map[string]interface{}
	if ctx, attributes, err = encoder.Decode(ctx, msg, model); err != nil {
		return apiserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("could not decode message: %w", err)
	}

	ok, err := h.callback.Consume(ctx, model, attributes)
	if err != nil {
		return apiserver.NewStatusResponse(http.StatusInternalServerError), fmt.Errorf("could not process model: %w", err)
	}

	if !ok {
		return apiserver.NewStatusResponse(http.StatusInternalServerError), fmt.Errorf("logical error: should not acknowledge model")
	}

	return apiserver.NewStatusResponse(http.StatusOK), nil
}
