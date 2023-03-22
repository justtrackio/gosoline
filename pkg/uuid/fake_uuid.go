package uuid

const FixedValueV4 = "00000000-0000-4000-a000-000000000000"

type FixedUuid struct{}

// NewV4 returns a static v4 string
func (u *FixedUuid) NewV4() string {
	return FixedValueV4
}

func NewFixedUuid() Uuid {
	return &FixedUuid{}
}
