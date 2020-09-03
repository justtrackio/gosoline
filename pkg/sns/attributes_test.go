package sns_test

import (
	"github.com/applike/gosoline/pkg/sns"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestIsValidAttributeName(t *testing.T) {
	valid := []string{
		"foo",
		"foo.bar",
		"FOO",
		"foo-bar.baz",
		"a",
		"foo-",
		"foo.aws",
		"a123-a",
		"5",
		"_",
		"_._",
		"-.-",
		"1-2",
		"this.is.reallyLOOOOOOOOOOOOOOOO00000000000NG.reallyLOOOOOOOOOOOOOOOO00000000000Ng",
		strings.Repeat("a", 256),
	}
	invalid := []string{
		"",
		".",
		"a..b",
		"aws.foo.bar",
		"amazoN.123",
		"foo.",
		".bar",
		"foo:bar",
		" foo.bar",
		"\000",
		strings.Repeat("a", 257),
	}

	for _, name := range valid {
		assert.True(t, sns.IsValidAttributeName(name), "expected %s to be a valid attribute name", name)
	}

	for _, name := range invalid {
		assert.False(t, sns.IsValidAttributeName(name), "expected %s to be an invalid attribute name", name)
	}
}
