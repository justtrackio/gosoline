package matcher

import (
	"context"

	"github.com/stretchr/testify/mock"
)

var Context = mock.MatchedBy(func(val any) bool {
	_, ok := val.(context.Context)

	return ok
})
