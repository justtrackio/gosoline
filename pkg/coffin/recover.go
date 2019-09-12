package coffin

import (
	"fmt"
	"github.com/pkg/errors"
)

func ResolveRecovery(unknownErr interface{}) error {
	switch rval := unknownErr.(type) {
	case nil:
		return nil

	case error:
		return rval

	case string:
		return errors.New(rval)

	default:
		return fmt.Errorf("unhandled error type %T", rval)
	}
}
