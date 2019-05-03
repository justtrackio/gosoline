package stream

import (
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileInput_Run(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := monMocks.NewLoggerMockedAll()

	input := NewFileInput(configMock, loggerMock, FileSettings{
		Filename: "testdata/file_input.json",
	})

	var err error
	go func() {
		err = input.Run()
	}()

	msg := <-input.Data()

	assert.Nil(t, err, "there should be no error in run")
	assert.Equal(t, "foobar", msg.Body, "the body should match")
}
