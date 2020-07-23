package coffin

import (
	"fmt"
)

func ResolveRecovery(unknownErr interface{}) error {
	switch rval := unknownErr.(type) {
	case nil:
		return nil

	case error:
		return rval

	case string:
		return fmt.Errorf(rval)

	default:
		return fmt.Errorf("unhandled error type %T", rval)
	}
}
