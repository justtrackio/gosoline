package stream

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
)

func TestIsFatalSqsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "QueueDoesNotExist error",
			err:      &types.QueueDoesNotExist{Message: aws.String("Queue does not exist")},
			expected: true,
		},
		{
			name:     "Access denied error",
			err:      fmt.Errorf("AccessDenied: Access to the resource is denied"),
			expected: true,
		},
		{
			name:     "Unauthorized operation error",
			err:      fmt.Errorf("UnauthorizedOperation: You are not authorized to perform this operation"),
			expected: true,
		},
		{
			name:     "Permission denied error",
			err:      fmt.Errorf("permission denied for queue"),
			expected: true,
		},
		{
			name:     "User not authorized error",
			err:      fmt.Errorf("user is not authorized to access this resource"),
			expected: true,
		},
		{
			name:     "Forbidden error",
			err:      fmt.Errorf("forbidden: insufficient permissions"),
			expected: true,
		},
		{
			name:     "Throttling error (recoverable)",
			err:      &types.RequestThrottled{Message: aws.String("Request was throttled")},
			expected: false,
		},
		{
			name:     "Network timeout error (recoverable)",
			err:      fmt.Errorf("network timeout"),
			expected: false,
		},
		{
			name:     "Generic error (recoverable)",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFatalSqsError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}