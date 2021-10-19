package uniqueid

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
)

type GeneratorMemory struct {
	sync.Mutex
	ids []int64
}

type generatorMemoryAppCtxKey string

func ProvideGeneratorMemory(ctx context.Context) (*GeneratorMemory, error) {
	generator, err := appctx.Provide(ctx, new(generatorMemoryAppCtxKey), func() (interface{}, error) {
		return &GeneratorMemory{
			Mutex: sync.Mutex{},
			ids:   make([]int64, 0),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return generator.(*GeneratorMemory), nil
}

// NewGeneratorMemory use this for integration tests
func NewGeneratorMemory(ctx context.Context) (Generator, error) {
	return ProvideGeneratorMemory(ctx)
}

func (g *GeneratorMemory) AddId(id int64) {
	g.Lock()
	defer g.Unlock()

	g.ids = append(g.ids, id)
}

func (g *GeneratorMemory) NextId(_ context.Context) (*int64, error) {
	g.Lock()
	defer g.Unlock()

	if len(g.ids) == 0 {
		return nil, fmt.Errorf("no unique ids left")
	}

	id := g.ids[0]
	g.ids = g.ids[1:]

	return &id, nil
}
