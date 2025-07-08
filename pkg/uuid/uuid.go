package uuid

import (
	"github.com/google/uuid"
)

//go:generate go run github.com/vektra/mockery/v2 --name Uuid
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

// Valid checks if the given string has a valid xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx uuid format
func ValidV4(s string) bool {
	if len(s) != 36 {
		return false
	}

	// it must be of the form  xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
	if s[8] != '-' || s[13] != '-' || s[14] != '4' || (s[19] != '8' && s[19] != '9' && s[19] != 'a' && s[19] != 'b') || s[18] != '-' || s[23] != '-' {
		return false
	}

	for _, x := range alfaNumIndexes {
		if !alfaNumChars[s[x]] {
			return false
		}
	}

	return true
}

var alfaNumChars = [256]bool{
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	true, true, true, true, true, true, true, true, true, true, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, true, true, true, true, true, true, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false,
}

var alfaNumIndexes = [32]int{
	0, 1, 2, 3, 4, 5, 6, 7,
	9, 10, 11, 12,
	14, 15, 16, 17,
	19, 20, 21, 22,
	24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35,
}
