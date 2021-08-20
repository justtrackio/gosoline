package uuid_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/applike/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRealUuidValidV4(t *testing.T) {
	uuidSource := uuid.New()

	for i := 0; i < 100; i++ {
		nextUuid := uuidSource.NewV4()
		lowerCaseUuid := strings.ToLower(nextUuid)
		upperCaseUuid := strings.ToUpper(nextUuid)
		assert.True(t, uuid.ValidV4(nextUuid), "Should be a valid uuid: %s", nextUuid)
		assert.True(t, uuid.ValidV4(lowerCaseUuid), "Should be a valid uuid: %s", lowerCaseUuid)
		assert.False(t, uuid.ValidV4(upperCaseUuid), "Should not a valid (lowercase) uuid: %s", upperCaseUuid)
		assert.Equal(t, nextUuid, lowerCaseUuid)
	}
}

func TestValidV4(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", false},
		{"not a uuid", false},
		{"00000000-0000-0000-0000-000000000000", false},
		{"d5b047878c18425b8c14e2c15d0e55de", false},
		{" d5b04787-8c18-425b-8c14-e2c15d0e55de", false},
		{"d5b04787-8c18-425b-8c14-e2c15d0e55de", true},
		{"00000000-0000-4000-8000-000000000000", true},
		{"d5b04787-8c18-425b-8c14-e2c15d0e55de", true},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			got := uuid.ValidV4(tt.input)

			assert.Equal(t, tt.want, got, fmt.Sprintf("uuid: %s", tt.input))
		})
	}
}
