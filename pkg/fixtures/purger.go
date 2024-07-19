package fixtures

import "context"

type Purger interface {
	Purge(ctx context.Context) error
}
