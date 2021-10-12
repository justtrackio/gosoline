package uniqueid

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type GeneratorMemory struct {
	sync.Mutex

	ids []int64
}

var gm = struct {
	sync.Mutex
	instance *GeneratorMemory
}{}

func ProvideGeneratorMemory() (*GeneratorMemory, error) {
	gm.Lock()
	defer gm.Unlock()

	if gm.instance != nil {
		return gm.instance, nil
	}

	gm.instance = &GeneratorMemory{
		Mutex: sync.Mutex{},
		ids:   make([]int64, 0),
	}

	return gm.instance, nil
}

func NewGeneratorMemory(_ context.Context, _ cfg.Config, _ log.Logger) (Generator, error) {
	return ProvideGeneratorMemory()
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
