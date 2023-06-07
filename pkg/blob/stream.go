package blob

import (
	"bytes"
	"io"
	"sync/atomic"

	"github.com/pkg/errors"
)

//go:generate mockery --name ReadCloser
type ReadCloser interface {
	io.ReadCloser
}

// A reader that we can close and that can seek
//
//go:generate mockery --name ReadSeekerCloser
type ReadSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

// A stream is a source of bytes you can either get as a full []byte or stream as a reader.
//
//go:generate mockery --name Stream
type Stream interface {
	// Read all data and close the reader.
	ReadAll() ([]byte, error)
	// Extract a reader you have to close yourself. Calling this multiple times might return
	// the same object.
	AsReader() ReadSeekerCloser
}

// Use a []byte as stream
func StreamBytes(data []byte) Stream {
	return byteStream{
		data: data,
	}
}

// Use a reader as a stream. If the reader does not implement Seek, we provide a dummy implementation.
func StreamReader(reader ReadCloser) Stream {
	if seeker, ok := reader.(ReadSeekerCloser); ok {
		return readerStream{
			reader: seeker,
		}
	}

	return noSeekerReaderStream{
		reader: reader,
	}
}

// CloseOnce wraps a reader and provide a closer. Can be called more than once. If the reader does not implement closer, it ignores calls to close.
func CloseOnce(reader io.ReadSeeker) ReadSeekerCloser {
	if closer, ok := reader.(ReadSeekerCloser); ok {
		return &onceCloser{
			closed: 0,
			reader: closer,
		}
	}

	return nopCloser{
		ReadSeeker: reader,
	}
}

// byte buffer based stream

type byteStream struct {
	data []byte
}

func (b byteStream) ReadAll() ([]byte, error) {
	return b.data, nil
}

func (b byteStream) AsReader() ReadSeekerCloser {
	return CloseOnce(bytes.NewReader(b.data))
}

// reader based stream

type readerStream struct {
	reader ReadSeekerCloser
}

func (r readerStream) ReadAll() ([]byte, error) {
	b, err := io.ReadAll(r.reader)

	if err == nil {
		err = r.reader.Close()
	}

	return b, err
}

func (r readerStream) AsReader() ReadSeekerCloser {
	return r.reader
}

// stream failing on seek

type noSeekerReaderStream struct {
	reader ReadCloser
}

func (r noSeekerReaderStream) ReadAll() ([]byte, error) {
	b, err := io.ReadAll(r.reader)

	if err == nil {
		err = r.reader.Close()
	}

	return b, err
}

// reader failing on seek

type fakeSeeker struct {
	ReadCloser
}

func (f fakeSeeker) Seek(_ int64, _ int) (int64, error) {
	return 0, errors.New("Not a real seeker")
}

func (r noSeekerReaderStream) AsReader() ReadSeekerCloser {
	return fakeSeeker{
		ReadCloser: r.reader,
	}
}

// a wrapper around a closer, which only calls close once on the wrapped reader

type onceCloser struct {
	// If 0, we haven't yet closed the reader
	// will be exchanged with 1 upon requesting close
	closed int32
	reader ReadSeekerCloser
}

func (o *onceCloser) Read(p []byte) (n int, err error) {
	return o.reader.Read(p)
}

func (o *onceCloser) Seek(offset int64, whence int) (int64, error) {
	return o.reader.Seek(offset, whence)
}

func (o *onceCloser) Close() error {
	if atomic.SwapInt32(&o.closed, 1) == 0 {
		return o.reader.Close()
	}

	return nil
}

// a wrapper, which provides Close() without doing anything

type nopCloser struct {
	io.ReadSeeker
}

func (o nopCloser) Close() error {
	return nil
}
