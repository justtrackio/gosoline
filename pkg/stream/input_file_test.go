package stream

import (
	"context"
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileInput_Run(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := logMocks.NewLoggerMockedAll()

	input := NewFileInput(configMock, loggerMock, FileSettings{
		Filename: "testdata/file_input.json",
	})

	var err error
	go func() {
		err = input.Run(context.Background())
	}()

	msg := <-input.Data()

	assert.Nil(t, err, "there should be no error in run")
	assert.Equal(t, "foobar", msg.Body, "the body should match")
}
