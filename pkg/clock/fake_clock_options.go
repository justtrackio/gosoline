package clock

type FakeClockOption func(*fakeClock)

func WithNonBlockingSleep(c *fakeClock) {
	c.nonBlockingSleep = true
}
