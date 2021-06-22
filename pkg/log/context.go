package log

import "context"

type key int

const contextFieldsKey key = 0

type ContextFieldsResolver func(ctx context.Context) map[string]interface{}

// NewLoggerContext returns a new Context carrying fields
func NewLoggerContext(ctx context.Context, fields map[string]interface{}) context.Context {
	return context.WithValue(ctx, contextFieldsKey, fields)
}

// AppendLoggerContextField appends the fields to the existing context fields, if there are no contextFields in the context
// then the value is initialized with the given fields.
// If there is a duplicate key then the newest value is the one that will be used.
func AppendLoggerContextField(ctx context.Context, fields map[string]interface{}) context.Context {
	contextFields := ContextLoggerFieldsResolver(ctx)
	contextFieldsLength := len(contextFields)

	if contextFieldsLength == 0 {
		return NewLoggerContext(ctx, fields)
	}

	newFields := make(map[string]interface{}, contextFieldsLength+len(fields))

	for k, v := range contextFields {
		newFields[k] = v
	}

	for k, v := range fields {
		newFields[k] = v
	}

	return NewLoggerContext(ctx, newFields)
}

// ContextLoggerFieldsResolver extracts the ContextFields from ctx, if not present returns empty ContextFields
func ContextLoggerFieldsResolver(ctx context.Context) map[string]interface{} {
	contextFields, ok := ctx.Value(contextFieldsKey).(map[string]interface{})

	if !ok {
		return map[string]interface{}{}
	}

	return contextFields
}
