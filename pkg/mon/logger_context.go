package mon

import "context"

type key int

const contextFieldsKey key = 0

// ContextFields is the type of the data that are stored in the context
type ContextFields map[string]interface{}

// NewLoggerContext returns a new Context carrying fields
func NewLoggerContext(ctx context.Context, fields ContextFields) context.Context {
	return context.WithValue(ctx, contextFieldsKey, fields)
}

// AppendLoggerContextField appends the fields to the existing context fields, if there are no contextFields in the context
// then the value is initialized with the given fields.
// If there is a duplicate key then the newest value is the one that will be used.
func AppendLoggerContextField(ctx context.Context, fields map[string]interface{}) context.Context {
	contextFields := fromLoggerContext(ctx)
	contextFieldsLength := len(contextFields)

	if contextFieldsLength == 0 {
		return NewLoggerContext(ctx, fields)
	}

	newFields := make(ContextFields, contextFieldsLength+len(fields))

	for k, v := range contextFields {
		newFields[k] = v
	}

	for k, v := range fields {
		newFields[k] = v
	}

	return NewLoggerContext(ctx, newFields)
}

// fromContextWithDefault extracts the ContextFields from ctx, if not present returns empty ContextFields
func fromLoggerContext(ctx context.Context) ContextFields {
	contextFields, ok := ctx.Value(contextFieldsKey).(ContextFields)
	if !ok {
		return ContextFields{}
	}

	return contextFields
}
