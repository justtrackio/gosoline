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
	attrVal = "val"
	attrTtl = "ttl"
)

type window struct {
	Key string `json:"key" ddb:"key=hash"`
	Val int    `json:"val"`
	Ttl int64  `json:"ttl" ddb:"ttl=enabled"`
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
	key := f.keyBuilder(prefix)

	item := &window{Key: key}

	now := f.clock.Now

	// This will create a new entry if none yet exists. It also will reset the current value to
	// 1 again, if time.Now is already past the entries TTL (which means, we are already in the next
	// time window and can release the lock). If none of those conditions apply for the current
	// state, it means that we can increment the value safely, because one of the following things must
	// be true:
	// 1. The TTL is still valid and the counter was not reset by another request -> we can increment
	// 2. The TTL has already expired between requests -> we might throttle too often -> we can increment
	// 3. The entry was already reset by another request -> we will return the new increment and ttl -> we are already
	//    in the next window -> we can increment
	reset := f.repo.UpdateItemBuilder().
		WithCondition(ddb.Or(
			ddb.AttributeNotExists(attrTtl),
			ddb.Lt(attrTtl, now().Unix()))).
		Set(attrVal, 1).
		SetIfNotExist(attrTtl, now().Add(f.window).Unix()).
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

	t := time.Duration(item.Ttl-now().Unix()) * time.Second
	return &item.Val, &t, nil
}

func (f fixedWindowDdb) keyBuilder(prefix string) string {
	return fmt.Sprintf("%s/%s", f.name, prefix)
}
