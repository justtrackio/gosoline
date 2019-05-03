package dispatcher

import "context"

type Result map[string]error
type Callback func(ctx context.Context, event interface{}) error

type Dispatcher interface {
	Fire(ctx context.Context, name string, event interface{}) Result
}

type dispatcher struct{}

var listeners = map[string][]Callback{}
var d = &dispatcher{}

func Get() *dispatcher {
	return d
}

func (d dispatcher) Fire(ctx context.Context, name string, event interface{}) Result {
	if _, ok := listeners[name]; !ok {
		return map[string]error{}
	}

	errors := make(Result)

	for _, c := range listeners[name] {
		errors[name] = c(ctx, event)
	}

	return errors
}

func On(name string, call Callback) {
	if _, ok := listeners[name]; !ok {
		listeners[name] = make([]Callback, 0)
	}

	listeners[name] = append(listeners[name], call)
}
