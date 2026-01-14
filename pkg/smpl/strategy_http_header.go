package smpl

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

const HeaderSamplingKey = "X-Goso-Sampled"

// DecideByHttpHeader creates a strategy based on the 'X-Goso-Sampled' HTTP header.
// It returns applied=true if the header is present, and parses the value as a boolean.
func DecideByHttpHeader(req *http.Request) Strategy {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		val := req.Header.Get(HeaderSamplingKey)

		if val == "" {
			return false, false, nil
		}

		if isSampled, err = strconv.ParseBool(val); err != nil {
			return false, false, fmt.Errorf("could not parse sampling header value '%s': %w", val, err)
		}

		return true, isSampled, nil
	}
}
