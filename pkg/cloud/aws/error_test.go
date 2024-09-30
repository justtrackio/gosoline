package aws_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestIsUsedClosedConnectionError(t *testing.T) {
	err := fmt.Errorf("something didn't work: %w", net.ErrClosed)

	isClosedErr := exec.IsUsedClosedConnectionError(err)
	assert.True(t, isClosedErr, "error: %v", err)
}
