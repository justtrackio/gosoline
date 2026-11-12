package guard

import "github.com/selm0/ladon"

// PolicySnapshot retains the exact stored value for conditional writes.
// Obtain snapshots through GetPolicySnapshot or GetSnapshot, not by constructing them.
type PolicySnapshot struct {
	Policy *ladon.DefaultPolicy
	id     string
	raw    string
}
