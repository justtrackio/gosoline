package stream

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInMemoryInput_Run(t *testing.T) {
	input := newInMemoryInput("myInput")

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
