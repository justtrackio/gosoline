package uuid

import (
	"github.com/google/uuid"
	"regexp"
)

var uuidRegExp = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

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

// Check if the given string is a valid lowercase UUID v4 string
func ValidV4(uuid string) bool {
	return uuidRegExp.Match([]byte(uuid))
}
