package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type privacySentryHub struct {
	scope *sentry.Scope
	err   error
}

func (h *privacySentryHub) ConfigureScope(f func(*sentry.Scope)) { f(h.scope) }
func (h *privacySentryHub) WithScope(f func(*sentry.Scope))      { f(h.scope) }
func (h *privacySentryHub) CaptureException(err error) *sentry.EventID {
	h.err = err

	return nil
}
func (*privacySentryHub) Flush(time.Duration) bool { return true }

func TestSentryCannotTraverseConfidentialCause(t *testing.T) {
	hub := &privacySentryHub{scope: sentry.NewScope()}
	handler := &HandlerSentry{hub: hub}
	cause := errors.New("provider echoed secret-473")
	protected := RedactError(cause, "secret-473")
	err := handler.Log(t.Context(), time.Now(), PriorityError, "", nil,
		fmt.Errorf("secret-473 authentication failed: %w", protected),
		Data{Fields: map[string]any{"api_key": "api-secret-473"}})
	require.NoError(t, err)
	assert.ErrorIs(t, protected, cause)
	assert.Nil(t, errors.Unwrap(hub.err))
	assert.NotContains(t, hub.err.Error(), "secret-473")
	event := hub.scope.ApplyToEvent(sentry.NewEvent(), nil, nil)
	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "api-secret-473")
}
