package stream

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/base64"
	"io"
)

const (
	gzipMaxExpectedBuffer = 8192
)

var (
	jsonArrayStart = []byte("[")
	jsonArraySep   = []byte(",")
	jsonArrayEnd   = []byte("]")
)

type producerDaemonAggregator struct {
	maxMessages  int
	maxBytes     int
	compression  CompressionType
	attributes   map[string]interface{}
	encodeBase64 bool

	buffer                   *bytes.Buffer
	writer                   io.WriteCloser
	messageCount             int
	uncompressedBytes        int
	expectedCompressionRatio float32
}

type AggregateFlush struct {
	Attributes   map[string]interface{}
	Body         string
	MessageCount int
}

//go:generate mockery -name ProducerDaemonAggregator
type ProducerDaemonAggregator interface {
	Write(msg *Message) ([]AggregateFlush, error)
	Flush() (*AggregateFlush, error)
}

func NewProducerDaemonAggregator(settings ProducerDaemonSettings, compression CompressionType) (ProducerDaemonAggregator, error) {
	a := &producerDaemonAggregator{
		maxMessages: settings.AggregationSize,
		maxBytes:    settings.AggregationMaxSize,
		compression: compression,
		attributes:  map[string]interface{}{},
		// initially assume we don't perform any compression, first message might not be packed as tightly as the rest,
		// but if your app runs for a little longer you will already have a proper ratio here for the second message
		expectedCompressionRatio: 1,
	}

	switch compression {
	case CompressionGZip:
		a.encodeBase64 = true
		// For gzip there might be some bytes in the buffer which are not yet written to our writer.
		// We expect that not more than this many bytes are still pending to be written when we have
		// written a record to the stream. It is most likely not much higher than 500 bytes, so we
		// have quite some headroom here.
		// We could also be calling Flush after each element, but that would break runs between messages
		// and also add additional bytes to encode the flush.
		if a.maxBytes > gzipMaxExpectedBuffer {
			a.maxBytes -= gzipMaxExpectedBuffer
		}
		a.attributes[AttributeCompression] = CompressionGZip
	case CompressionNone:
		a.encodeBase64 = false
	default:
		return nil, fmt.Errorf("unhandled compression type: %s", a.compression)
	}

	if err := a.reset(); err != nil {
		return nil, fmt.Errorf("failed to reset aggregate to initial state: %w", err)
	}

	return a, nil
}

func (a *producerDaemonAggregator) Write(msg *Message) ([]AggregateFlush, error) {
	encodedMessage, err := json.Marshal(msg)

	if err != nil {
		return nil, fmt.Errorf("failed to encode message for aggregate: %w", err)
	}

	expectedMessageSize := int(float32(len(encodedMessage)) * a.expectedCompressionRatio)

	var flushes []AggregateFlush
	if a.messageCount > 0 && a.maxBytes != 0 && a.getCurrentSize(expectedMessageSize) >= a.maxBytes {
		if flush, err := a.Flush(); err != nil {
			return nil, err
		} else {
			flushes = []AggregateFlush{*flush}
		}
	}

	if err := a.write(encodedMessage); err != nil {
		return nil, err
	}

	if a.messageCount >= a.maxMessages || (a.maxBytes != 0 && a.getCurrentSize(0) >= a.maxBytes) {
		if flush, err := a.Flush(); err != nil {
			return nil, err
		} else {
			flushes = append(flushes, *flush)
		}
	}

	return flushes, nil
}

func (a *producerDaemonAggregator) getCurrentSize(newMessageSize int) int {
	// estimate current size - we need to write at least the terminating ']' character
	currentSize := a.buffer.Len() + newMessageSize + 1
	if a.encodeBase64 {
		currentSize = currentSize * 4 / 3
	}

	return currentSize
}

func (a *producerDaemonAggregator) write(encodedMessage []byte) error {
	if a.messageCount > 0 {
		_, err := a.writer.Write(jsonArraySep)

		if err != nil {
			return fmt.Errorf("failed to write separator to buffer: %w", err)
		}
	}

	_, err := a.writer.Write(encodedMessage)

	if err != nil {
		return fmt.Errorf("failed to write message to buffer: %w", err)
	}

	a.messageCount++
	a.uncompressedBytes += len(encodedMessage) + 1

	return nil
}

func (a *producerDaemonAggregator) reset() error {
	a.messageCount = 0
	a.uncompressedBytes = 0

	if a.buffer == nil {
		// allocate a nice, large buffer at the start (128 kb should be enough to fit most messages as we normally limit
		// to 64 kb, so there is some headroom in the end)
		a.buffer = bytes.NewBuffer(make([]byte, 0, 128*1024))

		switch a.compression {
		case CompressionGZip:
			a.writer = gzip.NewWriter(a.buffer)
		case CompressionNone:
			a.writer = newWriterNopCloser(a.buffer)
		default:
			return fmt.Errorf("unhandled compression type: %s", a.compression)
		}
	} else {
		// re-use the buffer, we take care that we read its contents and convert it to a string (thereby copying it)
		// before we reset the aggregator, otherwise we will in the next step start to overwrite the data we already wrote
		a.buffer.Reset()

		switch a.compression {
		case CompressionGZip:
			a.writer.(*gzip.Writer).Reset(a.buffer)
		case CompressionNone:
			// nothing to do as we already reset the buffer
			break
		default:
			return fmt.Errorf("unhandled compression type: %s", a.compression)
		}
	}

	_, err := a.writer.Write(jsonArrayStart)
	a.uncompressedBytes += 1

	return err
}

type writerNopCloser struct {
	io.Writer
}

func (w writerNopCloser) Close() error {
	return nil
}

func newWriterNopCloser(writer io.Writer) io.WriteCloser {
	return writerNopCloser{
		Writer: writer,
	}
}

func (a *producerDaemonAggregator) Flush() (*AggregateFlush, error) {
	if _, err := a.writer.Write(jsonArrayEnd); err != nil {
		return nil, err
	}

	a.uncompressedBytes += 1

	// for gzip compression, close the writer so we write the footer, without compression this is a no-op
	if err := a.writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer during flush: %w", err)
	}

	messageCount := a.messageCount
	var body string
	if a.encodeBase64 {
		body = base64.EncodeToString(a.buffer.Bytes())
	} else {
		body = a.buffer.String()
	}

	// only update the expectation if we have some user data, if there are no messages in the aggregate, the ticker triggered
	// and we would otherwise expect a compression ration of much > 1.
	if messageCount > 0 {
		a.expectedCompressionRatio = float32(len(body)) / float32(a.uncompressedBytes)
	}

	if err := a.reset(); err != nil {
		return nil, err
	}

	return &AggregateFlush{
		Attributes:   a.attributes,
		MessageCount: messageCount,
		Body:         body,
	}, nil
}
