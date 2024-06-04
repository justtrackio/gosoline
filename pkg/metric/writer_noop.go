package metric

type noopWriter struct{}

func newNoopWriter() *noopWriter {
	return &noopWriter{}
}

func (n noopWriter) GetPriority() int {
	return 0
}

func (n noopWriter) Write(batch Data) {
}

func (n noopWriter) WriteOne(data *Datum) {
}
