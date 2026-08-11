package stream

import (
	"context"
	"sync"

	"github.com/justtrackio/gosoline/pkg/coffin"
)

var (
	inMemoryInputsLock sync.Mutex
	inMemoryInputs     = make(map[string]*InMemoryInput)
)

func ResetInMemoryInputs() {
	inMemoryInputsLock.Lock()
	defer inMemoryInputsLock.Unlock()

	for _, inp := range inMemoryInputs {
		inp.Reset()
	}
}

type InMemorySettings struct {
	Size        int `cfg:"size" default:"1"`
	RunnerCount int `cfg:"runner_count" default:"1" validate:"min=1"`
}

var _ Input = &InMemoryInput{}

type InMemoryInput struct {
	lck           sync.Mutex
	channel       chan *Message
	stopped       chan struct{}
	closedStopped bool
	settings      *InMemorySettings
}

func ProvideInMemoryInput(name string, settings *InMemorySettings) *InMemoryInput {
	inMemoryInputsLock.Lock()
	defer inMemoryInputsLock.Unlock()

	if input, ok := inMemoryInputs[name]; ok {
		return input
	}

	inMemoryInputs[name] = NewInMemoryInput(settings)

	return inMemoryInputs[name]
}

func NewInMemoryInput(settings *InMemorySettings) *InMemoryInput {
	if settings.RunnerCount <= 0 {
		settings.RunnerCount = 1
	}

	return &InMemoryInput{
		channel:       make(chan *Message, settings.Size),
		stopped:       make(chan struct{}),
		closedStopped: false,
		settings:      settings,
	}
}

func (i *InMemoryInput) Reset() {
	i.lck.Lock()
	defer i.lck.Unlock()

	i.channel = make(chan *Message, i.settings.Size)
	i.stopped = make(chan struct{})
	i.closedStopped = false
}

func (i *InMemoryInput) Publish(messages ...*Message) {
	for _, msg := range messages {
		i.lck.Lock()
		channel := i.channel
		stopped := i.stopped
		i.lck.Unlock()

		select {
		case channel <- msg:
		case <-stopped:
			return
		}
	}
}

func (i *InMemoryInput) Run(ctx context.Context, process InputProcess) error {
	i.lck.Lock()
	stopped := i.stopped
	channel := i.channel
	i.lck.Unlock()

	cfn := coffin.New()

	for range i.settings.RunnerCount {
		cfn.Gof(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-stopped:
					// messages which got published before the input was stopped still have to be processed.
					// as soon as both the stopped channel and the message channel are ready, select would
					// pick one of them at random and we would drop those messages.
					drainInMemoryInput(ctx, channel, process)

					return nil
				case <-cfn.Dying():
					return nil
				case msg := <-channel:
					process(ctx, msg)
				}
			}
		}, "panic in in-memory input runner")
	}

	return cfn.Wait()
}

// drainInMemoryInput processes all messages which are currently buffered in the given channel. It returns as soon
// as the channel is empty or the context got canceled.
func drainInMemoryInput(ctx context.Context, channel <-chan *Message, process InputProcess) {
	for {
		// check the context first, otherwise select would randomly keep draining a canceled context
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case msg := <-channel:
			process(ctx, msg)
		default:
			return
		}
	}
}

func (i *InMemoryInput) Stop(ctx context.Context) {
	i.lck.Lock()
	defer i.lck.Unlock()

	if !i.closedStopped {
		close(i.stopped)
		i.closedStopped = true
	}
}

func (i *InMemoryInput) IsHealthy() bool {
	return true
}
