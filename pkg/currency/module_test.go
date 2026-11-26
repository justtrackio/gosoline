package currency

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/require"
)

// leaseProvider models expiration and token-checked renewal and release. A stale
// release is a no-op, so it cannot hide a missing renewal error in the caller.
type leaseProvider struct {
	mu           sync.Mutex
	next         int
	leases       map[string]leaseState
	afterAcquire func(context.Context)
	beforeRenew  func(context.Context) error
}

type leaseState struct {
	token   int
	expires time.Time
}

type testLease struct {
	provider *leaseProvider
	resource string
	token    int
}

func (p *leaseProvider) Acquire(ctx context.Context, resource string) (conc.DistributedLock, error) {
	p.mu.Lock()

	if err := ctx.Err(); err != nil {
		p.mu.Unlock()
		return nil, err
	}
	if lease, ok := p.leases[resource]; ok && time.Now().Before(lease.expires) {
		p.mu.Unlock()
		return nil, conc.ErrLockOwned
	}
	p.next++
	if p.leases == nil {
		p.leases = make(map[string]leaseState)
	}
	p.leases[resource] = leaseState{token: p.next, expires: time.Now().Add(3 * time.Minute)}
	lock := &testLease{provider: p, resource: resource, token: p.next}
	p.mu.Unlock()
	if p.afterAcquire != nil {
		p.afterAcquire(ctx)
	}
	return lock, nil
}

func (p *leaseProvider) TryAcquireIn(ctx context.Context, resource string, _ time.Duration) (conc.DistributedLock, error) {
	return p.Acquire(ctx, resource)
}

// steal simulates an independent owner replacing the current token.
func (p *leaseProvider) steal(resource string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	p.leases[resource] = leaseState{token: p.next, expires: time.Now().Add(3 * time.Minute)}
}

func (l *testLease) Renew(ctx context.Context, duration time.Duration) error {
	if l.provider.beforeRenew != nil {
		if err := l.provider.beforeRenew(ctx); err != nil {
			return err
		}
	}

	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	lease, ok := l.provider.leases[l.resource]
	if !ok || lease.token != l.token || !time.Now().Before(lease.expires) {
		return conc.ErrLockNotOwned
	}
	lease.expires = time.Now().Add(duration)
	l.provider.leases[l.resource] = lease
	return nil
}

func (l *testLease) Release() error {
	l.provider.mu.Lock()
	defer l.provider.mu.Unlock()

	lease, ok := l.provider.leases[l.resource]
	if ok && lease.token == l.token && time.Now().Before(lease.expires) {
		delete(l.provider.leases, l.resource)
	}
	return nil
}

func TestUpdateExchangeRatesMaintainsLeaseUntilRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		provider := &leaseProvider{}
		module := Module{lockProvider: provider}
		resource := lockResourceRecentExchangeRates
		finish := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			result <- module.updateExchangeRates(ctx, resource, func(ctx context.Context) error {
				select {
				case <-finish:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		}()

		synctest.Wait()
		time.Sleep(4 * time.Minute)
		synctest.Wait()
		_, err := provider.Acquire(ctx, resource)
		require.ErrorIs(t, err, conc.ErrLockOwned, "another owner must not start while refreshing")

		close(finish)
		require.NoError(t, <-result, "refresh should finish and release its lease")
		other, err := provider.Acquire(ctx, resource)
		require.NoError(t, err, "lease should be available after refresh")
		require.NoError(t, other.Release())
	})
}

func TestUpdateExchangeRatesConfirmsSlowAcquire(t *testing.T) {
	for _, tc := range []struct {
		name    string
		delay   time.Duration
		expired bool
	}{
		{name: "renew before refresh", delay: 2 * time.Minute},
		{name: "expired before refresh", delay: 3 * time.Minute, expired: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				provider := &leaseProvider{afterAcquire: func(context.Context) { time.Sleep(tc.delay) }}
				module := Module{lockProvider: provider}
				started := make(chan struct{})
				finish := make(chan struct{})
				defer close(finish)
				result := make(chan error, 1)
				go func() {
					result <- module.updateExchangeRates(t.Context(), lockResourceRecentExchangeRates, func(context.Context) error {
						close(started)
						<-finish
						return nil
					})
				}()

				synctest.Wait()
				time.Sleep(tc.delay)
				synctest.Wait()
				if tc.expired {
					select {
					case <-started:
						t.Fatal("refresh started after its acquired lease expired")
					default:
					}
					select {
					case err := <-result:
						require.ErrorIs(t, err, conc.ErrLockNotOwned)
					default:
						t.Fatal("expired acquire did not fail")
					}
					return
				}

				select {
				case <-started:
				default:
					t.Fatal("refresh did not start after confirming ownership")
				}
				// The original lease expires at three minutes, so a successor
				// must still be excluded beyond that point.
				time.Sleep(90 * time.Second)
				synctest.Wait()
				_, err := provider.Acquire(t.Context(), lockResourceRecentExchangeRates)
				require.ErrorIs(t, err, conc.ErrLockOwned)
			})
		})
	}
}

func TestUpdateExchangeRatesReportsInFlightRenewalTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		renewStarted := make(chan struct{})
		provider := &leaseProvider{beforeRenew: func(ctx context.Context) error {
			close(renewStarted)
			<-ctx.Done()
			return ctx.Err()
		}}
		module := Module{lockProvider: provider}
		finish := make(chan struct{})
		result := make(chan error, 1)
		go func() {
			result <- module.updateExchangeRates(t.Context(), lockResourceRecentExchangeRates, func(context.Context) error {
				<-finish
				return nil
			})
		}()

		synctest.Wait()
		time.Sleep(time.Minute)
		<-renewStarted
		close(finish)
		synctest.Wait()
		select {
		case err := <-result:
			t.Fatalf("returned before the in-flight renewal completed: %v", err)
		default:
		}
		_, err := provider.Acquire(t.Context(), lockResourceRecentExchangeRates)
		require.ErrorIs(t, err, conc.ErrLockOwned, "lease must be held until renewal completes")

		time.Sleep(time.Minute)
		synctest.Wait()
		select {
		case err := <-result:
			require.ErrorIs(t, err, context.DeadlineExceeded)
		default:
			t.Fatal("renewal timeout was not reported")
		}
		other, err := provider.Acquire(t.Context(), lockResourceRecentExchangeRates)
		require.NoError(t, err, "lease should be released after renewal completes")
		require.NoError(t, other.Release())
	})
}

func TestUpdateExchangeRatesLetsModuleRecoverProviderPanic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		module := Module{lockProvider: &leaseProvider{}}
		require.PanicsWithValue(t, "provider panic", func() {
			require.NoError(t, module.updateExchangeRates(t.Context(), lockResourceRecentExchangeRates, func(context.Context) error {
				panic("provider panic")
			}))
		})
	})
}

// historicalOnlyUpdater lets Run reach the real historical updater without
// making the recent refresh depend on provider or store behavior.
type historicalOnlyUpdater struct{ UpdaterService }

func (u historicalOnlyUpdater) EnsureRecentExchangeRates(context.Context) error { return nil }

type recordingStore struct {
	kvstore.KvStore[float64]
	writes int
}

func (s *recordingStore) Get(context.Context, any, *float64) (bool, error) { return false, nil }
func (s *recordingStore) Put(context.Context, any, float64) error {
	s.writes++
	return nil
}

func (s *recordingStore) PutBatch(context.Context, any) error {
	s.writes++
	return nil
}

type historicalTestProvider struct {
	fetch func(context.Context, []time.Time) (map[time.Time][]Rate, error)
}

func (p historicalTestProvider) Name() string                                     { return "test" }
func (p historicalTestProvider) GetPriority() int                                 { return 0 }
func (p historicalTestProvider) FetchLatestRates(context.Context) ([]Rate, error) { return nil, nil }

func (p historicalTestProvider) FetchHistoricalRates(ctx context.Context, dates []time.Time) (map[time.Time][]Rate, error) {
	return p.fetch(ctx, dates)
}

func TestRunStopsHistoricalRefreshWhenLeaseIsLost(t *testing.T) {
	for _, returnSuccess := range []bool{false, true} {
		name := "provider returns cancellation"
		if returnSuccess {
			name = "provider returns stale success"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				leases := &leaseProvider{}
				store := &recordingStore{}
				started := make(chan struct{})
				canceled := make(chan struct{})
				fallbackCalled := false
				provider := historicalTestProvider{fetch: func(ctx context.Context, dates []time.Time) (map[time.Time][]Rate, error) {
					close(started)
					<-ctx.Done()
					close(canceled)
					if returnSuccess {
						return map[time.Time][]Rate{dates[0]: {{Currency: "USD", Rate: 1.25}}}, nil
					}
					return nil, ctx.Err()
				}}
				fallback := historicalTestProvider{fetch: func(context.Context, []time.Time) (map[time.Time][]Rate, error) {
					fallbackCalled = true
					return nil, nil
				}}
				now := time.Date(2021, 5, 27, 0, 0, 0, 0, time.UTC)
				logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
				updater := NewUpdaterWithInterfaces(logger, store, nil, clock.NewFakeClockAt(now), &Settings{StartDate: now}, []Provider{provider, fallback})
				module := Module{logger: logger, updaterService: historicalOnlyUpdater{updater}, healthy: &atomic.Bool{}, lockProvider: leases}
				result := make(chan error, 1)
				go func() { result <- module.Run(ctx) }()

				<-started
				synctest.Wait()
				leases.steal(lockResourceHistoricalExchangeRates)
				time.Sleep(time.Minute)
				synctest.Wait()
				select {
				case <-canceled:
				default:
					t.Fatal("provider context was not canceled when ownership was lost")
				}
				select {
				case err := <-result:
					require.ErrorIs(t, err, conc.ErrLockNotOwned)
				default:
					t.Fatal("Run did not return after ownership was lost")
				}
				require.False(t, fallbackCalled, "historical fallback must stop after cancellation")
				require.Zero(t, store.writes, "stale refresh must not write rates or freshness markers")
				_, err := leases.Acquire(ctx, lockResourceHistoricalExchangeRates)
				require.ErrorIs(t, err, conc.ErrLockOwned, "stale release must leave the new owner's lease intact")
			})
		})
	}
}
