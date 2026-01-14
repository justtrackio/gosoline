package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
)

type ctxKey string

func init() {
	// Register a custom sampling strategy that can be referenced in config.
	//
	// This is process-global and should be done before application startup, before a decider is created.
	smpl.AddStrategy("force-by-context", func(ctx context.Context, config cfg.Config) (smpl.Strategy, error) {
		return func(ctx context.Context) (applied bool, sampled bool, err error) {
			if v, ok := ctx.Value(ctxKey("force_sample")).(bool); ok {
				return true, v, nil
			}

			return false, false, nil
		}, nil
	})
}

func main() {
	ctx := context.Background()
	config := cfg.New()
	logger := log.NewLogger()

	// Pretend this is coming from an incoming request/message.
	ctx = context.WithValue(ctx, ctxKey("force_sample"), false)

	// Build a decider from config. It reads `sampling.enabled` and `sampling.strategies`.
	// In real applications you typically use smpl.ProvideDecider(ctx, config).
	decider, err := smpl.NewDecider(ctx, config)
	if err != nil {
		logger.Error(ctx, "can not create decider: %w", err)

		return
	}

	// Decide applies the configured strategies.
	ctx, sampled, err := decider.Decide(ctx)
	if err != nil {
		logger.Error(ctx, "can not decide: %w", err)

		return
	}

	fmt.Printf("config decision: sampled=%v (smplctx.IsSampled=%v)\n", sampled, smplctx.IsSampled(ctx))

	// Per-call overrides: additional strategies run before the configured strategies.
	// This allows you to force sampling behaviour for specific code paths.
	alwaysStrategy, err := smpl.DecideByAlways(ctx, config)
	if err != nil {
		logger.Error(ctx, "can not build override strategy: %w", err)

		return
	}

	ctx, sampled, err = decider.Decide(ctx, alwaysStrategy)
	if err != nil {
		logger.Error(ctx, "can not decide with override: %w", err)

		return
	}

	fmt.Printf("override decision: sampled=%v (smplctx.IsSampled=%v)\n", sampled, smplctx.IsSampled(ctx))
}
