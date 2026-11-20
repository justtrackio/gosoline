package env

// MmockComponent provides the address of the mock server.
type MmockComponent struct {
	baseComponent
	address string
}

// Address returns the base URL for application requests to the mock server.
func (c *MmockComponent) Address() string {
	return c.address
}
