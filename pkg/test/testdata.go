package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func ReadTestdataString(t *testing.T, filename string) string {
	bytes, err := ioutil.ReadFile(filename)

	if err != nil {
		err = fmt.Errorf("can not read test data from file %s: %w", filename, err)
		assert.Fail(t, err.Error())
		return ""
	}

	return string(bytes)
}
