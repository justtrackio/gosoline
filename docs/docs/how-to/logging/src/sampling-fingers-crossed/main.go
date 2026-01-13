package main

import (
	"context"
	"fmt"
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
)

func main() {
	// 1. Create a logger with sampling enabled
	handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityDebug, log.FormatterConsole, "main", "15:04:05.000", os.Stdout)
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	if err := logger.Option(log.WithSamplingEnabled(true)); err != nil {
		panic(err)
	}

	// 2. Prepare a context that is NOT sampled (to trigger buffering)
	// In a real app, this decision comes from the sampling middleware or decider.
	ctx := context.Background()
	ctx = smplctx.WithSampling(ctx, smplctx.Sampling{Sampled: false})

	// 3. Add the fingers-crossed scope to the context
	// This scope will buffer logs until an error occurs or it is flushed.
	ctx = log.WithFingersCrossedScope(ctx)

	fmt.Println("--- Phase 1: Logging Info (buffered, should not appear yet) ---")
	logger.Info(ctx, "This is an info message (buffered)")
	logger.Debug(ctx, "This is a debug message (buffered)")

	fmt.Println("--- Phase 2: Logging Error (triggers flush) ---")
	// This error log will cause the buffer to flush, printing the previous messages + the error.
	logger.Error(ctx, "An error occurred!")

	fmt.Println("--- Phase 3: Done ---")
}
