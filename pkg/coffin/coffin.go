package coffin

import (
	"context"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
)

type Coffin interface {
	Alive() bool
	Context(parent context.Context) context.Context
	Dead() <-chan struct{}
	Dying() <-chan struct{}
	Err() (reason error)
	Go(f func() error)
	Gof(f func() error, name string, args ...interface{})
	Kill(reason error)
	Killf(f string, a ...interface{}) error
	Wait() error
}

type coffin struct {
	tomb.Tomb
}

func New() Coffin {
	return &coffin{}
}

func (c *coffin) Go(f func() error) {
	c.Tomb.Go(func() (err error) {
		defer func() {
			err = ResolveRecovery(recover())
		}()

		return f()
	})
}

func (c *coffin) Gof(f func() error, msg string, args ...interface{}) {
	c.Tomb.Go(func() (err error) {
		defer func() {
			err = ResolveRecovery(recover())
			err = errors.Wrapf(err, msg, args...)
		}()

		return f()
	})
}

func WithContext(parent context.Context) (Coffin, context.Context) {
	tmb, ctx := tomb.WithContext(parent)
	cfn := &coffin{Tomb: *tmb}

	return cfn, ctx
}
