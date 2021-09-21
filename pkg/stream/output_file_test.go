package stream_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sync"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

func TestOutputFile_ConcurrentWrite(t *testing.T) {
	fileName := "testdata/output_file_test.output.txt"

	_ = os.Remove(fileName)

	logger := new(logMocks.Logger)
	output := stream.NewFileOutput(nil, logger, &stream.FileOutputSettings{
		Filename: fileName,
		Append:   true,
	})
	var waitGroup sync.WaitGroup
	count := 10
	waitGroup.Add(count)

	for i := 0; i < count; i++ {
		go func(i int) {
			defer waitGroup.Done()
			err := output.WriteOne(context.Background(), &stream.Message{
				Attributes: nil,
				Body:       fmt.Sprintf("%d", i),
			})
			assert.NoError(t, err)
		}(i)
	}

	waitGroup.Wait()

	result, err := ioutil.ReadFile(fileName)
	assert.NoError(t, err)
	assert.False(t, regexp.MustCompile("\n{2}").Match(result), "unexpected double new line")
}
