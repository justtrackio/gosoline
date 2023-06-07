package env

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type filesystem struct {
	t *testing.T
}

func newFilesystem(t *testing.T) *filesystem {
	return &filesystem{
		t: t,
	}
}

func (f *filesystem) ReadString(filename string) string {
	bytes, err := os.ReadFile(filename)

	if err != nil {
		err = fmt.Errorf("can not read test data from file %s: %w", filename, err)
		assert.FailNow(f.t, err.Error())
		return ""
	}

	return string(bytes)
}
