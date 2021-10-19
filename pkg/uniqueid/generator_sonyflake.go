package uniqueid

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/sony/sonyflake"
)

type GeneratorSonyFlakeSettings struct {
	StartTime time.Time `cfg:"start_time" default:"2021-09-02T00:00:00Z"`
	MachineId uint16    `cfg:"machine_id"`
}

type generatorSonyFlake struct {
	sonyFlake *sonyflake.Sonyflake
}

// NewGeneratorSonyFlake use this to generate unique ids in a distributed fashion.
// The id consists of a timestamp (10ms resolution), a unique machine id and a sequence number which is atomically
// increased per machine. It's recommended to get the machine id by parsing the lower 16 bits of your ip.
// For more info: https://github.com/sony/sonyflake
func NewGeneratorSonyFlake(config cfg.Config, _ log.Logger) (Generator, error) {
	settings := &GeneratorSonyFlakeSettings{}
	config.UnmarshalKey("unique_id", settings)

	generator := sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: settings.StartTime,
		MachineID: func() (uint16, error) {
			return settings.MachineId, nil
		},
	})

	if generator == nil {
		return nil, fmt.Errorf("could not create sonyflake generator")
	}

	return &generatorSonyFlake{
		sonyFlake: generator,
	}, nil
}

func (g *generatorSonyFlake) NextId(_ context.Context) (*int64, error) {
	id, err := g.sonyFlake.NextID()
	if err != nil {
		return nil, err
	}

	return mdl.Int64(int64(id)), nil
}
