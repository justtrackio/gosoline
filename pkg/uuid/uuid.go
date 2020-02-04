package uuid

import (
	"github.com/google/uuid"
)

//go:generate mockery -name Uuid
type Uuid interface {
	NewV4() string
}

type RealUuid struct{}

func New() Uuid {
	return &RealUuid{}
}

// NewV4 returns a UUID v4 string
func (u *RealUuid) NewV4() string {
	return uuid.New().String()
}
