package mdlsub

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type Handler struct {
	callback stream.ConsumerCallback
	tracer   tracing.Tracer
}

func NewHandler(ctx context.Context, config cfg.Config, logger log.Logger, callbackFactory stream.ConsumerCallbackFactory) (httpserver.HandlerWithInput, error) {
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

func (h *Handler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	msg := request.Body.(*stream.Message)

	var err error
	var model interface{}

	if model = h.callback.GetModel(msg.Attributes); model == nil {
		err := fmt.Errorf("invalid or incomplete attributes: %v", msg.Attributes)
		return httpserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("could not get model: %w", err)
	}

	encoding := stream.GetEncodingAttribute(msg.Attributes)

	if encoding == nil {
		return httpserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("missing encoding attribute")
	}

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: *encoding,
	})

	var attributes map[string]string
	if ctx, attributes, err = encoder.Decode(ctx, msg, model); err != nil {
		return httpserver.NewStatusResponse(http.StatusBadRequest), fmt.Errorf("could not decode message: %w", err)
	}

	ok, err := h.callback.Consume(ctx, model, attributes)
	if err != nil {
		return httpserver.NewStatusResponse(http.StatusInternalServerError), fmt.Errorf("could not process model: %w", err)
	}

	if !ok {
		return httpserver.NewStatusResponse(http.StatusInternalServerError), fmt.Errorf("logical error: should not acknowledge model")
	}

	return httpserver.NewStatusResponse(http.StatusOK), nil
}
