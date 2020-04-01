package stream

import (
	"context"
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInMemoryInput_Run(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := monMocks.NewLoggerMockedAll()

	input := newInMemoryInputFromConfig(configMock, loggerMock, "myInput")

	// language=JSON
	message := NewJsonMessage(`{ "foo" : "bar" }`, map[string]interface{}{})

	SendToInMemoryInput("myInput", message)

	var err error
	go func() {
		err = input.Run(context.Background())
	}()

	msg := <-input.Data()

	assert.NoError(t, err)
	assert.Equal(t, message, msg)
}
