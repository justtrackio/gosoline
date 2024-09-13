package fixtures

import (
	"crypto/sha256"
	"io"
)

type ByteSequence interface {
	io.Reader
	// GetBytes returns a slice of n bytes filled with random-looking data.
	GetBytes(n int) []byte
}

type byteSequence struct {
	nextSeed []byte
}

// NewByteSequence provides a way to generate random-looking values for fixtures from a given seed. A possible use case would be to provide the name
// of the fixture set you are creating as the seed to get distinct values from other fixture sets, but having deterministic values compared to just
// generating random bytes for example.
func NewByteSequence(seed string) ByteSequence {
	return &byteSequence{
		nextSeed: []byte(seed),
	}
}

func (h *byteSequence) getNextHash() []byte {
	hash := sha256.Sum256(h.nextSeed)
	h.nextSeed = hash[:]

	return hash[:]
}

func (h *byteSequence) read(p []byte) {
	for i := 0; i < len(p); i += 32 {
		hash := h.getNextHash()
		if i+32 <= len(p) {
			copy(p[i:], hash)
		} else {
			copy(p[i:], hash[:len(p)-i])
		}
	}
}

func (h *byteSequence) GetBytes(n int) []byte {
	result := make([]byte, n)
	h.read(result)

	return result
}

func (h *byteSequence) Read(p []byte) (n int, err error) {
	h.read(p)

	return len(p), nil
}
