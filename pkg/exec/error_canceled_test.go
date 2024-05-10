package exec_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestIsRequestCanceled(t *testing.T) {
	ctxForCancel, cancelCtx := context.WithCancel(t.Context())
	cancelCtx()

	ctxForDeadlineCanceled, cancelDeadline := context.WithTimeout(t.Context(), time.Hour)
	cancelDeadline()

	ctxForDeadlineExpired, cancelExpiredDeadline := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancelExpiredDeadline()
	<-ctxForDeadlineExpired.Done()

	for name, test := range map[string]struct {
		err        error
		isCanceled bool
	}{
		"none": {
			err:        nil,
			isCanceled: false,
		},
		"other error": {
			err:        io.EOF,
			isCanceled: false,
		},
		"format error": {
			err:        fmt.Errorf("error: %d", 42),
			isCanceled: false,
		},
		"simple": {
			err:        context.Canceled,
			isCanceled: true,
		},
		"simple wrapped": {
			err:        fmt.Errorf("error %w", context.Canceled),
			isCanceled: true,
		},
		"exec": {
			err:        exec.RequestCanceledError,
			isCanceled: true,
		},
		"exec wrapped": {
			err:        fmt.Errorf("error %w", exec.RequestCanceledError),
			isCanceled: true,
		},
		"multierror empty": {
			err:        multierror.Append(nil),
			isCanceled: false,
		},
		"multierror single": {
			err:        multierror.Append(nil, context.Canceled),
			isCanceled: true,
		},
		"multierror single with nil": {
			err:        multierror.Append(nil, context.Canceled, nil),
			isCanceled: true,
		},
		"multierror single wrapped": {
			err:        multierror.Append(nil, fmt.Errorf("error %w", context.Canceled)),
			isCanceled: true,
		},
		"multierror multiple wrapped": {
			err:        multierror.Append(nil, fmt.Errorf("error %w", context.Canceled), fmt.Errorf("error %w", exec.RequestCanceledError)),
			isCanceled: true,
		},
		"multierror mixed": {
			err:        multierror.Append(nil, context.Canceled, io.EOF),
			isCanceled: false,
		},
		"multierror mixed swapped": {
			err:        multierror.Append(nil, io.EOF, context.Canceled),
			isCanceled: false,
		},
		"from canceled context": {
			err:        ctxForCancel.Err(),
			isCanceled: true,
		},
		"from canceled deadline": {
			err:        ctxForDeadlineCanceled.Err(),
			isCanceled: true,
		},
		"from expired deadline": {
			err:        ctxForDeadlineExpired.Err(),
			isCanceled: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.isCanceled, exec.IsRequestCanceled(test.err), "Expected canceled = %v for error %v", test.isCanceled, test.err)
		})
	}
}
