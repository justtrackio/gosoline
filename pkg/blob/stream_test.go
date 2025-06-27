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
	closerImpl := mocks.NewReadSeekerCloser(t)
	closerImpl.EXPECT().Close().Return(nil).Once()

	closer := blob.CloseOnce(closerImpl)

	// close many times
	for i := 0; i < 10; i++ {
		err := closer.Close()
		assert.NoError(t, err)
	}
}

func TestCloseOnceConcurrently(t *testing.T) {
	closerImpl := mocks.NewReadSeekerCloser(t)
	closerImpl.EXPECT().Close().Return(nil).Once()

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
	reader := mocks.NewReadSeekerCloser(t)

	stream := blob.StreamReader(reader)

	assert.Equal(t, reader, stream.AsReader())

	data := []byte("my data")

	reader.EXPECT().Read(mock.MatchedBy(func(arg any) bool {
		_, ok := arg.([]byte)

		return ok
	})).Run(func(buffer []byte) {
		assert.GreaterOrEqual(t, len(buffer), len(data))
		copy(buffer, data)

		reader.EXPECT().Read(mock.MatchedBy(func(arg any) bool {
			_, ok := arg.([]byte)

			return ok
		})).Return(0, io.EOF).Once()
	}).Return(len(data), nil).Once()
	reader.EXPECT().Close().Return(nil).Once()

	bytes, err := stream.ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, data, bytes)
}

func TestStreamReaderWithoutSeek(t *testing.T) {
	reader := mocks.NewReadCloser(t)

	stream := blob.StreamReader(reader)
	streamReader := stream.AsReader()

	buffer := make([]byte, 1)

	reader.EXPECT().Read(buffer).Return(1, nil)
	n, err := streamReader.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = streamReader.Seek(0, io.SeekCurrent)
	assert.Error(t, err)

	reader.EXPECT().Close().Return(nil).Once()
	err = streamReader.Close()
	assert.NoError(t, err)

	// create the reader again to restore its state
	stream = blob.StreamReader(reader)

	data := []byte("my data")

	reader.EXPECT().Read(mock.MatchedBy(func(arg any) bool {
		_, ok := arg.([]byte)

		return ok
	})).Run(func(buffer []byte) {
		assert.GreaterOrEqual(t, len(buffer), len(data))
		copy(buffer, data)

		reader.EXPECT().Read(mock.MatchedBy(func(arg any) bool {
			_, ok := arg.([]byte)

			return ok
		})).Return(0, io.EOF).Once()
	}).Return(len(data), nil).Once()
	reader.EXPECT().Close().Return(nil).Once()

	bytes, err := stream.ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, data, bytes)
}
