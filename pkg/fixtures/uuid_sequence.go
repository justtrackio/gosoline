package fixtures

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/uuid"
)

type UuidSequence interface {
	uuid.Uuid
}

type uuidSequence struct {
	byteSequence ByteSequence
}

// NewUuidSequence provides a way to generate random-looking UUIDs for fixtures from a given seed.
// Use this for fixtures instead of generating random UUIDs to create distinct keys easily while
// also keeping the fixtures deterministic. A unique seed is required for each fixture set to ensure
// there are no collisions between the generated UUIDs.
func NewUuidSequence(seed string) UuidSequence {
	return uuidSequence{
		byteSequence: NewByteSequence(seed),
	}
}

func (s uuidSequence) NewV4() string {
	uuidBytes := s.byteSequence.GetBytes(16)

	// set version and variant
	uuidBytes[6] = (uuidBytes[6] & 0x0F) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3F) | 0x80

	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		uuidBytes[0],
		uuidBytes[1],
		uuidBytes[2],
		uuidBytes[3],
		uuidBytes[4],
		uuidBytes[5],
		uuidBytes[6],
		uuidBytes[7],
		uuidBytes[8],
		uuidBytes[9],
		uuidBytes[10],
		uuidBytes[11],
		uuidBytes[12],
		uuidBytes[13],
		uuidBytes[14],
		uuidBytes[15],
	)
}
