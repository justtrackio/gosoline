package assert

import (
	"bufio"
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"os"
	"testing"
)

type message map[string]interface{}
type messages []message

type outputFile struct {
	messages messages
}

func ReadOutputFile(path string) *outputFile {
	messages := readMessagesFromFile(path)

	return &outputFile{
		messages: messages,
	}
}

func (a *outputFile) Index(i int) message {
	return a.messages[i]
}

func (a *outputFile) Len() int {
	return len(a.messages)
}

func MessageBodyJsonEqual(t *testing.T, m message, path string, expected interface{}, msg string) {
	body := m["body"].(string)

	actual := gjson.Get(body, path)
	assert.Equal(t, expected, actual.Value(), msg)
}

func MessageAttributesJsonEqual(t *testing.T, m message, name string, expected interface{}, msg string) {
	attributes := m["attributes"].(map[string]interface{})
	assert.Equal(t, expected, attributes[name], msg)
}

func readMessagesFromFile(path string) messages {
	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Errorf("could not open file with messages: %s", err.Error()))
	}

	messages := make(messages, 0)

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		line := scan.Text()
		message := make(message)

		err = json.Unmarshal([]byte(line), &message)

		if err != nil {
			panic(fmt.Errorf("could not unmarshal line with message: %s", err.Error()))
		}

		messages = append(messages, message)
	}

	if err := scan.Err(); err != nil {
		panic(fmt.Errorf("could not read lines from output file: %s", err.Error()))
	}

	err = file.Close()
	if err != nil {
		panic(fmt.Errorf("could not close file with messages: %s", err.Error()))
	}

	return messages
}
