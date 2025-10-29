package mapx_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/stretchr/testify/assert"
)

func TestSplitUnescapedDotN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want []string
	}{
		// Basic behavior
		{"simple no limit", "a.b.c", -1, []string{"a", "b", "c"}},
		{"simple with limit", "a.b.c", 2, []string{"a", "b.c"}},
		{"simple n=1", "a.b.c", 1, []string{"a.b.c"}},
		{"simple n=0", "a.b.c", 0, nil},

		// Escaped dots
		{"escaped middle no limit", "a\\.b.c", -1, []string{"a\\.b", "c"}},
		{"escaped middle limited", "a\\.b.c.d", 3, []string{"a\\.b", "c", "d"}},
		{"escaped dot only", "a\\.b\\.c", -1, []string{"a\\.b\\.c"}},
		{"escaped at start", "\\.a.b", -1, []string{"\\.a", "b"}},
		{"escaped at end", "a.b\\.", -1, []string{"a", "b\\."}},

		// Mix escaped + unescaped
		{"mixed escapes", "a\\.b.c\\.d.e", -1, []string{"a\\.b", "c\\.d", "e"}},
		{"mixed escapes limited", "a\\.b.c\\.d.e", 2, []string{"a\\.b", "c\\.d.e"}},

		// Edge structure
		{"no dots", "abc", -1, []string{"abc"}},
		{"leading dot", ".a.b", -1, []string{"", "a", "b"}},
		{"trailing dot", "a.b.", -1, []string{"a", "b", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapx.SplitUnescapedDotN(tt.in, tt.n)
			assert.Equal(t, tt.want, got)
		})
	}
}
