package appctx

import (
	"context"
	"fmt"
	"sync"
)

type containerKeyType int

var containerKey containerKeyType = 1

type ErrNoApplicationContainerFound struct{}

func (e ErrNoApplicationContainerFound) Error() string {
	return "no application container found in context"
}

type ErrNoItemFound struct {
	Key interface{}
}

func (e ErrNoItemFound) Error() string {
	return fmt.Sprintf("no item with key %v found", e.Key)
}

type container struct {
	items sync.Map
}

func WithContainer(ctx context.Context) context.Context {
	return context.WithValue(ctx, containerKey, &container{
		items: sync.Map{},
	})
}

func Provide(ctx context.Context, key interface{}, factory func() (interface{}, error)) (interface{}, error) {
	var ok bool
	var err error
	var contI, val interface{}

	if contI = ctx.Value(containerKey); contI == nil {
		return nil, &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	if val, ok = cont.items.Load(key); ok {
		return val, nil
	}

	if val, err = factory(); err != nil {
		return nil, err
	}

	cont.items.Store(key, val)

	return val, nil
}

func CopyContainer(from, to context.Context) (context.Context, error) {
	var contI interface{}

	if contI = from.Value(containerKey); contI == nil {
		return nil, &ErrNoApplicationContainerFound{}
	}

	return context.WithValue(to, containerKey, contI), nil
}
