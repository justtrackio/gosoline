package blob_test

import (
	"io"
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/blob/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCloseOnce(t *testing.T) {
	closerImpl := new(mocks.ReadSeekerCloser)
	closerImpl.On("Close").Return(nil).Once()

	closer := blob.CloseOnce(closerImpl)

	// close many times
	for i := 0; i < 10; i++ {
		err := closer.Close()
		assert.NoError(t, err)
	}

	closerImpl.AssertExpectations(t)
}

func TestCloseOnceConcurrently(t *testing.T) {
	closerImpl := new(mocks.ReadSeekerCloser)
	closerImpl.On("Close").Return(nil).Once()

	closer := blob.CloseOnce(closerImpl)

	var wg sync.WaitGroup
	wg.Add(100)

	mutex := sync.RWMutex{}
	mutex.Lock()

	// close many times in different threads
	for i := 0; i < 100; i++ {
		go func() {
			// wait until all threads are created and can start concurrently
			mutex.RLock()

			err := closer.Close()
			assert.NoError(t, err)

			mutex.RUnlock()
			wg.Done()
		}()
	}

	// unleash threads
	mutex.Unlock()

	// wait for threads
	wg.Wait()

	closerImpl.AssertExpectations(t)
}

func TestStreamBytes(t *testing.T) {
	bytes := []byte("these are my bytes")

	stream := blob.StreamBytes(bytes)

	// we should be able to read this more than once
	for i := 0; i < 10; i++ {
		read, err := stream.ReadAll()

		assert.NoError(t, err)
		assert.Equal(t, bytes, read)
	}

	reader := stream.AsReader()

	buf := make([]byte, 1)
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, bytes[:1], buf)

	pos, err := reader.Seek(0, io.SeekCurrent)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pos)

	err = reader.Close()
	assert.NoError(t, err)
}

func TestStreamReaderWithSeek(t *testing.T) {
	reader := new(mocks.ReadSeekerCloser)

	stream := blob.StreamReader(reader)

	assert.Equal(t, reader, stream.AsReader())

	data := []byte("my data")

	reader.On("Read", mock.Anything).Run(func(args mock.Arguments) {
		buffer := args.Get(0).([]byte)

		assert.GreaterOrEqual(t, len(buffer), len(data))

		copy(buffer, data)

		reader.On("Read", mock.Anything).Return(0, io.EOF).Once()
	}).Return(len(data), nil).Once()
	reader.On("Close").Return(nil).Once()

	bytes, err := stream.ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, data, bytes)

	reader.AssertExpectations(t)
}

func TestStreamReaderWithoutSeek(t *testing.T) {
	reader := new(mocks.ReadCloser)

	stream := blob.StreamReader(reader)
	streamReader := stream.AsReader()

	buffer := make([]byte, 1)

	reader.On("Read", buffer).Return(1, nil)
	n, err := streamReader.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = streamReader.Seek(0, io.SeekCurrent)
	assert.Error(t, err)

	reader.On("Close").Return(nil).Once()
	err = streamReader.Close()
	assert.NoError(t, err)

	reader.AssertExpectations(t)

	// create the reader again to restore its state
	stream = blob.StreamReader(reader)

	data := []byte("my data")

	reader.On("Read", mock.Anything).Run(func(args mock.Arguments) {
		buffer := args.Get(0).([]byte)

		assert.GreaterOrEqual(t, len(buffer), len(data))

		copy(buffer, data)

		reader.On("Read", mock.Anything).Return(0, io.EOF).Once()
	}).Return(len(data), nil).Once()
	reader.On("Close").Return(nil).Once()

	bytes, err := stream.ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, data, bytes)

	reader.AssertExpectations(t)
}
