package conc_test

import (
	"context"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/conc/mocks"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// simple happy path

func TestHoldDistributedLock_Simple(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(nil).Times(2)

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 350)

	lock.AssertExpectations(t)
	lock.On("Release").Return(nil).Once()

	err := release()
	assert.NoError(t, err)

	lock.AssertExpectations(t)
}

// context canceled after 200ms

func TestHoldDistributedLock_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	lock := new(mocks.DistributedLock)
	lock.On("Renew", ctx, time.Millisecond*300).Return(nil).Once()

	release := conc.HoldDistributedLock(ctx, lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 200)

	lock.AssertExpectations(t)
	lock.On("Release").Return(nil).Once()

	cancel()

	// give the worker time to trigger renew again should it still be running
	time.Sleep(time.Millisecond * 200)

	err := release()
	assert.NoError(t, err)

	lock.AssertExpectations(t)
}

// renew fails after 1 try

func TestHoldDistributedLock_RenewFails(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(nil).Once()

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 200)

	lock.AssertExpectations(t)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(conc.NotOwnedError).Once()
	time.Sleep(time.Millisecond * 400)

	lock.AssertExpectations(t)
	lock.On("Release").Return(nil).Once()

	err := release()
	assert.Error(t, err)
	assert.Equal(t, conc.NotOwnedError, err)

	lock.AssertExpectations(t)
}

// renew fails and release also fails

func TestHoldDistributedLock_RenewAndReleaseFails(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(conc.NotOwnedError).Once()

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 400)

	lock.AssertExpectations(t)
	lock.On("Release").Return(conc.AlreadyPoisonedErr).Once()

	err := release()
	assert.Error(t, err)
	assert.Equal(t, multierror.Append(conc.NotOwnedError, conc.AlreadyPoisonedErr), err)

	lock.AssertExpectations(t)
}

// renew fails with a canceled context

func TestHoldDistributedLock_RenewFailsWithCanceled(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(context.Canceled).Once()

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 400)

	lock.AssertExpectations(t)
	lock.On("Release").Return(nil).Once()

	err := release()
	assert.NoError(t, err)

	lock.AssertExpectations(t)
}

// renew fails with a canceled context and renew also fails

func TestHoldDistributedLock_RenewFailsWithCanceledAndReleaseFails(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(context.Canceled).Once()

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 400)

	lock.AssertExpectations(t)
	lock.On("Release").Return(conc.AlreadyPoisonedErr).Once()

	err := release()
	assert.Error(t, err)
	assert.Equal(t, conc.AlreadyPoisonedErr, err)

	lock.AssertExpectations(t)
}

// release fails

func TestHoldDistributedLock_ReleaseFails(t *testing.T) {
	lock := new(mocks.DistributedLock)
	lock.On("Renew", context.Background(), time.Millisecond*300).Return(nil).Once()

	release := conc.HoldDistributedLock(context.Background(), lock, time.Millisecond*300)
	time.Sleep(time.Millisecond * 200)

	lock.AssertExpectations(t)
	lock.On("Release").Return(conc.AlreadyPoisonedErr).Once()

	err := release()
	assert.Error(t, err)
	assert.Equal(t, conc.AlreadyPoisonedErr, err)

	lock.AssertExpectations(t)
}
