package metric

type multiWriter struct {
	writers []Writer
}

func NewMultiWriterWithInterfaces(writers []Writer) Writer {
	return &multiWriter{
		writers: writers,
	}
}

func (m *multiWriter) GetPriority() int {
	prio := PriorityLow

	for _, w := range m.writers {
		wPriority := w.GetPriority()
		if wPriority > prio {
			prio = wPriority
		}
	}

	return prio
}

func (m *multiWriter) Write(batch Data) {
	for _, w := range m.writers {
		w.Write(batch)
	}
}

func (m *multiWriter) WriteOne(datum *Datum) {
	for _, w := range m.writers {
		w.WriteOne(datum)
	}
}
