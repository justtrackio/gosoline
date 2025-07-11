package kinesis

import "fmt"

type StreamBusyError struct {
	stream Stream
}

func NewStreamBusyError(stream Stream) error {
	return &StreamBusyError{
		stream: stream,
	}
}

func (e *StreamBusyError) Error() string {
	return fmt.Sprintf("Stream is busy: %s", e.stream)
}

func (e *StreamBusyError) As(target any) bool {
	if err, ok := target.(*StreamBusyError); ok && err != nil {
		*err = *e

		return true
	}

	return false
}

type NoSuchStreamError struct {
	stream Stream
}

func NewNoSuchStreamError(stream Stream) error {
	return &NoSuchStreamError{
		stream: stream,
	}
}

func (e *NoSuchStreamError) Error() string {
	return fmt.Sprintf("No such stream: %s", e.stream)
}

func (e *NoSuchStreamError) As(target any) bool {
	if err, ok := target.(*NoSuchStreamError); ok && err != nil {
		*err = *e

		return true
	}

	return false
}
