package email

import (
	"encoding/base64"
	"io"
)

func writeMIMEBase64(writer io.Writer, data []byte) error {
	wrappedWriter := &mimeBase64Writer{writer: writer}
	encoder := base64.NewEncoder(base64.StdEncoding, wrappedWriter)
	if _, err := encoder.Write(data); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if wrappedWriter.column == 0 {
		return nil
	}

	return writeMIMELineBreak(writer)
}

func writeMIMELineBreak(writer io.Writer) error {
	written, err := io.WriteString(writer, mimeLineBreak)
	if err != nil {
		return err
	}
	if written != len(mimeLineBreak) {
		return io.ErrShortWrite
	}

	return nil
}

type mimeBase64Writer struct {
	writer io.Writer
	column int
}

func (w *mimeBase64Writer) Write(data []byte) (int, error) {
	// RFC 2045 section 6.8 limits MIME base64 lines to 76 characters.
	const lineLength = 76

	written := 0
	for len(data) > 0 {
		length := min(lineLength-w.column, len(data))
		n, err := w.writer.Write(data[:length])
		written += n
		if err != nil {
			return written, err
		}
		if n != length {
			return written, io.ErrShortWrite
		}

		w.column += n
		data = data[n:]
		if w.column == lineLength {
			if err := writeMIMELineBreak(w.writer); err != nil {
				return written, err
			}
			w.column = 0
		}
	}

	return written, nil
}
