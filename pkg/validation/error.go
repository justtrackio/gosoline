package validation

import (
	"fmt"
	"strings"
)

type Error struct {
	Errors []error
}

func (e *Error) Error() string {
	messages := make([]string, len(e.Errors))
	for i := 0; i < len(e.Errors); i++ {
		messages[i] = e.Errors[i].Error()
	}

	return fmt.Sprintf("validation: %s", strings.Join(messages, "; "))
}

func (e *Error) Is(err error) bool {
	_, ok := err.(*Error)

	return ok
}

func (e *Error) As(target interface{}) bool {
	targetErr, ok := target.(*Error)

	if ok {
		*targetErr = *e
	}

	return ok
}
