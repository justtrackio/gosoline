package appctx

import "context"

type Option func(ctx context.Context) error
