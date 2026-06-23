package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

func apiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	tracer, err := tracing.ProvideTracer(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("could not provide tracer: %w", err)
	}

	definitions := &httpserver.Definitions{}
	definitions.GET("/hello", httpserver.CreateHandler(&HelloHandler{
		logger: logger,
		tracer: tracer,
		mw:     metric.NewWriter(),
	}))

	return definitions, nil
}

type HelloHandler struct {
	logger log.Logger
	tracer tracing.Tracer
	mw     metric.Writer
}

func (h *HelloHandler) GetInput() any {
	return nil
}

func (h *HelloHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	_, span := h.tracer.StartSubSpan(ctx, "hello-work")
	defer span.Finish()

	h.mw.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "HelloRequests",
		Value:      1.0,
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})

	time.Sleep(10 * time.Millisecond)

	h.logger.Info(ctx, "handled hello request")

	return httpserver.NewJsonResponse(map[string]string{
		"message": fmt.Sprintf("Hello from OTel example at %s", time.Now().Format(time.RFC3339)),
	}), nil
}

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithTracing,
		application.WithMetrics,
		application.WithModuleFactory("api", httpserver.New("default", apiDefiner)),
	)
}
