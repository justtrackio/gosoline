package limit

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	attrVal             = "val"
	attrTtl             = "ttl"
	attrWindowExpiresAt = "windowExpiresAt"
)

type window struct {
	Key string `json:"key" ddb:"key=hash"`
	Val int    `json:"val"`
	// Ttl is the cleanup expiry in Unix seconds, rounded up.
	Ttl int64 `json:"ttl" ddb:"ttl=enabled"`
	// WindowExpiresAt is the precise window expiry in Unix nanoseconds.
	WindowExpiresAt int64 `json:"windowExpiresAt"`
}

func (w window) expiresAt() time.Time {
	if w.WindowExpiresAt != 0 {
		return time.Unix(0, w.WindowExpiresAt)
	}

	// Legacy windows expire when ttl < now.Unix(): the following second.
	return time.Unix(w.Ttl+1, 0)
}

type fixedWindowDdb struct {
	logger log.Logger
	clock  clock.Clock
	repo   ddb.Repository
	tRepo  ddb.TransactionRepository
	window time.Duration
	name   string
}

func NewFixedWindowDdb(ctx context.Context, config cfg.Config, logger log.Logger, settings *ddb.Settings, c FixedWindowConfig) (LimiterWithMiddleware, error) {
	if c.Window <= 0 {
		return nil, fmt.Errorf("fixed window duration must be positive, got %s", c.Window)
	}

	settings.Main.Model = &window{}

	repo, err := ddb.NewRepository(ctx, config, logger, settings)
	if err != nil {
		return nil, err
	}

	transactionRepo, err := ddb.NewTransactionRepository(ctx, config, logger, "default")
	if err != nil {
		return nil, err
	}

	backend := &fixedWindowDdb{
		logger: logger.WithChannel("rate_limiter_incrementer_ddb"),
		clock:  clock.NewRealClock(),
		repo:   repo,
		tRepo:  transactionRepo,
		window: c.Window,
		name:   c.Name,
	}

	builder, err := newInvocationBuilder(c.Name)
	if err != nil {
		return nil, err
	}

	return NewFixedWindowLimiter(backend, clock.NewRealClock(), c, builder), nil
}

func (f fixedWindowDdb) Increment(ctx context.Context, prefix string) (incr *int, ttl *time.Duration, err error) {
	item := &window{Key: f.keyBuilder(prefix)}

	now := f.clock.Now()
	expiresAt := now.Add(f.window)
	// Cleanup must not delete a window before its precise expiry.
	cleanupTtl := expiresAt.Unix()
	if expiresAt.Nanosecond() != 0 {
		cleanupTtl++
	}

	// Existing rows keep their legacy expiry until their first renewal.
	legacyExpired := ddb.And(
		ddb.AttributeNotExists(attrWindowExpiresAt),
		ddb.Or(ddb.AttributeNotExists(attrTtl), ddb.Lt(attrTtl, now.Unix())),
	)
	expired := ddb.Or(ddb.Lte(attrWindowExpiresAt, now.UnixNano()), legacyExpired)

	reset := f.repo.UpdateItemBuilder().
		WithCondition(expired).
		Set(attrVal, 1).
		Set(attrWindowExpiresAt, expiresAt.UnixNano()).
		Set(attrTtl, cleanupTtl).
		ReturnAllNew()

	resp, err := f.repo.UpdateItem(ctx, reset, item)
	if err != nil {
		return nil, nil, err
	}

	if resp.ConditionalCheckFailed {
		increment := f.repo.UpdateItemBuilder().
			Add(attrVal, 1).
			ReturnAllNew()

		_, err := f.repo.UpdateItem(ctx, increment, item)
		if err != nil {
			return nil, nil, err
		}
	}

	remaining := item.expiresAt().Sub(f.clock.Now())

	return &item.Val, &remaining, nil
}

func (f fixedWindowDdb) keyBuilder(prefix string) string {
	return fmt.Sprintf("%s/%s", f.name, prefix)
}
