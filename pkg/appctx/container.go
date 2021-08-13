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
	sync.Mutex
	items map[interface{}]interface{}
}

func WithContainer(ctx context.Context) context.Context {
	return context.WithValue(ctx, containerKey, &container{
		items: make(map[interface{}]interface{}),
	})
}

func Set(ctx context.Context, key interface{}, value interface{}) error {
	contI := ctx.Value(containerKey)

	if contI == nil {
		return &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	cont.Lock()
	defer cont.Unlock()

	cont.items[key] = value

	return nil
}

func Get(ctx context.Context, key interface{}) (interface{}, error) {
	contI := ctx.Value(containerKey)

	if contI == nil {
		return nil, &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	cont.Lock()
	defer cont.Unlock()

	if val, ok := cont.items[key]; ok {
		return val, nil
	}

	return nil, &ErrNoItemFound{
		Key: key,
	}
}

func GetSet(ctx context.Context, key interface{}, factory func() (interface{}, error)) (interface{}, error) {
	var ok bool
	var err error
	var contI, val interface{}

	if contI = ctx.Value(containerKey); contI == nil {
		return factory()
		// return nil, &ErrNoApplicationContainerFound{}
	}

	cont := contI.(*container)

	cont.Lock()
	defer cont.Unlock()

	if val, ok = cont.items[key]; ok {
		return val, nil
	}

	if val, err = factory(); err != nil {
		return nil, err
	}

	cont.items[key] = val

	return val, nil
}
