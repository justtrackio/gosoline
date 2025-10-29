package env

import "testing"

type tLogWriter struct {
	t *testing.T
}

func (w tLogWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("%s", p)
w.t.Failed()
	return 0, nil
}
