package env

// MmockComponent provides the addresses of the mock server and its admin API.
type MmockComponent struct {
	baseComponent
	address      string
	adminAddress string
}

// Address returns the base URL for application requests to the mock server.
func (c *MmockComponent) Address() string {
	return c.address
}

// AdminAddress returns the base URL for Mmock's administrative API.
func (c *MmockComponent) AdminAddress() string {
	return c.adminAddress
}
