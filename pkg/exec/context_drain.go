package exec

import "context"

type drainContextKey struct{}

// WithDrainContext attaches drainCtx to ctx to communicate the deadline for in-flight processing.
//
// It exists so that a component which hands work to a callback does not have to guess how long that callback may still
// run once a shutdown started: the owner of the callback attaches its own drain context and thereby defines the single
// authoritative processing deadline. Components finding a drain context must honour this contract:
//
//   - They must not start a grace timer of their own for processing. Two independent timers over the same piece of work
//     lead to the work being abandoned by one side while the other still trusts its result.
//   - They must keep in-flight processing alive until the drain context is done, even if their own context is canceled
//     earlier.
//   - They may start their own commit or acknowledgement window once the drain context is done, because that is the
//     point in time at which no further processing can happen.
//
// Components finding no drain context must propagate cancellation immediately: without an attached deadline the caller
// owns cancellation, and inventing a grace period would silently extend a shutdown the caller expected to be immediate.
func WithDrainContext(ctx context.Context, drainCtx context.Context) context.Context {
	return context.WithValue(ctx, drainContextKey{}, drainCtx)
}

// DrainContextFrom returns the drain context attached to ctx by WithDrainContext and whether there was one. See
// WithDrainContext for the contract a caller has to fulfill when a drain context is present.
func DrainContextFrom(ctx context.Context) (context.Context, bool) {
	drainCtx, ok := ctx.Value(drainContextKey{}).(context.Context)

	return drainCtx, ok
}
