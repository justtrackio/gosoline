package log

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/funk"
)

type (
	key    int
	fields struct {
		lck  *sync.RWMutex
		data map[string]any
	}
	contextFields struct {
		localFields  fields
		globalFields fields
	}
)

const contextFieldsKey key = 0

type ContextFieldsResolverFunction func(ctx context.Context) map[string]any

func newContext(ctx context.Context, localFields fields, globalFields fields) context.Context {
	return context.WithValue(ctx, contextFieldsKey, contextFields{
		localFields:  localFields,
		globalFields: globalFields,
	})
}

// InitContext returns a new context capable of carrying (mutable) local and global logger fields.
//
// ---------------------------------------------------------------------------------------------------------------------
//
// A note about local and global context fields:
//   - A local field is private to the current application. It will NOT be attached to stream.Messages or transferred in
//     some other way to another service. Thus, it is somewhat cheaper to add local fields to a context as they will only
//     not affect any downstream applications or increase message size.
//   - A global field is the inverse of a local field. It will be forwarded with a message and thus "infect" other services.
//     For some fields this might be desirable, for example, to record the identity of the user of an event. You should
//     however be careful when adding many global fields as they will increase the size of any encoded message.
//
// When logging messages, global fields overwrite local fields if they share the same name.
//
// ---------------------------------------------------------------------------------------------------------------------
//
// A note about mutability:
// After using any of the methods returning a new context from this file (InitContext, AppendContextFields,
// MutateContextFields, AppendGlobalContextFields, MutateGlobalContextFields), the returned context will be mutable.
// This means that you can add context fields using MutateContextFields or MutateGlobalContextFields to the existing
// context (even though a new context is returned, it will be the same context if the context was already mutable). This
// allows for the following pattern:
//
//	func caller(ctx context.Context, logger log.Logger, input int) {
//		ctx = log.InitContext(ctx)
//
//		if result, err := callee(ctx, input); err == nil {
//			logger.WithContext(ctx).Info("Computed result %d", result)
//		} else {
//			logger.WithContext(ctx).Error("Failed to compute result: %w", err)
//		}
//	}
//
//	func callee(ctx context.Context, input int) (int, error) {
//		_ = log.MutateContextFields(ctx, map[string]any{
//			"input_value": input,
//		})
//
//		if input < 10 {
//			return 0, fmt.Errorf("input must not be smaller than 10")
//		}
//
//		return input - 10, nil
//	}
//
// After the callee returns, we add the (now mutated) context to the logger and will see the original input value in the
// logged fields even though it was for example not included in the error we got back.
func InitContext(ctx context.Context) context.Context {
	return appendContextFields(ctx, nil, true, true)
}

// AppendContextFields appends the fields to the existing local context, creating a new context containing the
// merged fields.
//
// Any existing fields with the same key as any new field provided will be overwritten.
//
// This method breaks mutation links for the local context only. If you mutate global fields on the returned context and
// the original context was already mutable, it will see these changes as well. Example:
//
//	func mutateFields() {
//		ctx := log.InitContext(context.Background())
//		localCtx := log.AppendContextFields(ctx, map[string]any{
//			"field": "value",
//		})
//
//		// mutations of local fields are only propagated to the local context
//		localCtx = log.MutateContextFields(localCtx, map[string]any{
//			"field": "new_value",
//		})
//		localFields := log.LocalOnlyContextFieldsResolver(localCtx)
//		print(localFields["field"]) // "new_value"
//		// the parent context wasn't changed
//		localFields = log.LocalOnlyContextFieldsResolver(ctx)
//		print(len(localFields)) // 0
//
//		// mutations of global fields are still propagated to the original context
//		localCtx = log.MutateGlobalContextFields(localCtx, map[string]any{
//			"other_field": "other_value",
//		})
//		globalFields := log.GlobalContextFieldsResolver(ctx)
//		print(globalFields["other_field"]) // "other_value"
//	}
func AppendContextFields(ctx context.Context, newFields map[string]any) context.Context {
	return appendContextFields(ctx, newFields, true, false)
}

// MutateContextFields is similar to AppendContextFields, but it mutates the fields from the context
// if the context already contains fields which can be mutated. Otherwise, it initializes a new context able to carry
// fields in the future.
func MutateContextFields(ctx context.Context, newFields map[string]any) context.Context {
	return appendContextFields(ctx, newFields, true, true)
}

// AppendGlobalContextFields is similar to AppendContextFields, but appends to global fields instead.
func AppendGlobalContextFields(ctx context.Context, newFields map[string]any) context.Context {
	return appendContextFields(ctx, newFields, false, false)
}

// MutateGlobalContextFields is similar to MutateContextFields, but mutates to global fields instead.
func MutateGlobalContextFields(ctx context.Context, newFields map[string]any) context.Context {
	return appendContextFields(ctx, newFields, false, true)
}

func appendContextFields(ctx context.Context, newFields map[string]any, local bool, mutate bool) context.Context {
	value, ok := ctx.Value(contextFieldsKey).(contextFields)
	if !ok {
		localFields := fields{
			lck:  &sync.RWMutex{},
			data: make(map[string]any),
		}
		globalFields := fields{
			lck:  &sync.RWMutex{},
			data: make(map[string]any),
		}

		updateFields := &globalFields
		if local {
			updateFields = &localFields
		}

		// make a copy of the input data as we don't know if it will be mutated later on
		updateFields.data = funk.MergeMaps(newFields)

		return newContext(ctx, localFields, globalFields)
	}

	updateFields := value.globalFields
	if local {
		updateFields = value.localFields
	}

	if mutate {
		updateFields.lck.Lock()
		defer updateFields.lck.Unlock()

		for k, v := range newFields {
			updateFields.data[k] = v
		}

		return ctx
	}

	updateFields.lck.RLock()
	defer updateFields.lck.RUnlock()

	updatedFields := fields{
		lck:  &sync.RWMutex{},
		data: funk.MergeMaps(updateFields.data, newFields),
	}

	if local {
		value.localFields = updatedFields
	} else {
		value.globalFields = updatedFields
	}

	return newContext(ctx, value.localFields, value.globalFields)
}

// LocalOnlyContextFieldsResolver extracts the local fields from a context and, if not present, it returns an empty map.
//
// Warning: Besides very specific circumstances this method is most likely not what you want. Consider using
// GlobalContextFieldsResolver or ContextFieldsResolver instead. Outside of tests there shouldn't normally be a need to
// ignore any configured global fields from a context.
func LocalOnlyContextFieldsResolver(ctx context.Context) map[string]any {
	return contextFieldsResolver(ctx, true, false)
}

// GlobalContextFieldsResolver extracts the global fields from a context and, if not present, it returns an empty map.
func GlobalContextFieldsResolver(ctx context.Context) map[string]any {
	return contextFieldsResolver(ctx, false, true)
}

// ContextFieldsResolver extracts the local and global fields from a context and, if not present, it returns an empty map.
//
// Global fields overwrite local fields of the same name.
func ContextFieldsResolver(ctx context.Context) map[string]any {
	return contextFieldsResolver(ctx, true, true)
}

func contextFieldsResolver(ctx context.Context, local bool, global bool) map[string]any {
	value, ok := ctx.Value(contextFieldsKey).(contextFields)

	if !ok {
		return map[string]any{}
	}

	var fields []map[string]any

	if local {
		// be careful about the locking in this method. We always have to lock the local fields first before locking the
		// global fields (if we lock both in a method). Otherwise, we have an opportunity for a deadlock.
		value.localFields.lck.RLock()
		defer value.localFields.lck.RUnlock()

		fields = append(fields, value.localFields.data)
	}

	// need to append the global data second to ensure we honor our comment about
	// global fields taking precedence over local fields
	if global {
		value.globalFields.lck.RLock()
		defer value.globalFields.lck.RUnlock()

		fields = append(fields, value.globalFields.data)
	}

	// make a copy of the data to ensure we can mutate the maps in another goroutine
	return funk.MergeMaps(fields...)
}
