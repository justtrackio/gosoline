package stream

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileInput_Run(t *testing.T) {
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	filename := writeFileInput(t, `{"body":"one"}
{"body":"two"}
`)

	input := NewFileInput(nil, loggerMock, FileSettings{
		Filename: filename,
	})

	var messages []string
	err := input.Run(t.Context(), func(_ context.Context, msg *Message) bool {
		messages = append(messages, msg.Body)

		return true
	})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"one", "two"}, messages)
}

func TestFileInput_RunConcurrently(t *testing.T) {
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	filename := writeFileInput(t, `{"body":"one"}
{"body":"two"}
{"body":"three"}
`)
	input := NewFileInput(nil, loggerMock, FileSettings{Filename: filename, RunnerCount: 2})

	var lock sync.Mutex
	active := 0
	maxActive := 0
	err := input.Run(t.Context(), func(_ context.Context, _ *Message) bool {
		lock.Lock()
		active++
		maxActive = max(maxActive, active)
		lock.Unlock()

		time.Sleep(10 * time.Millisecond)

		lock.Lock()
		active--
		lock.Unlock()

		return true
	})

	require.NoError(t, err)
	assert.Equal(t, 2, maxActive)
}

func TestFileInput_SkipsInvalidMessages(t *testing.T) {
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	filename := writeFileInput(t, `{"body":"one"}
invalid
{"body":"two"}
`)
	input := NewFileInput(nil, loggerMock, FileSettings{Filename: filename})

	var messages []string
	err := input.Run(t.Context(), func(_ context.Context, msg *Message) bool {
		messages = append(messages, msg.Body)

		return true
	})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"one", "two"}, messages)
}

func TestFileInput_Stop(t *testing.T) {
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	filename := writeFileInput(t, `{"body":"one"}
{"body":"two"}
`)
	input := NewFileInput(nil, loggerMock, FileSettings{Filename: filename})

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	var once sync.Once
	go func() {
		done <- input.Run(t.Context(), func(_ context.Context, _ *Message) bool {
			once.Do(func() {
				close(started)
			})
			<-release

			return true
		})
	}()

	<-started
	input.Stop(t.Context())
	close(release)

	require.NoError(t, <-done)
}

func writeFileInput(t *testing.T, contents string) string {
	t.Helper()

	filename := t.TempDir() + "/input.json"
	require.NoError(t, os.WriteFile(filename, []byte(contents), 0o600))

	return filename
}
