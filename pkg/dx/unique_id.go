package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

const (
	defaultMachineId     = uint16(1)
	defaultGeneratorType = "sonyflake"
)

func init() {
	cfg.AddPostProcessor(10, "gosoline.dx.uniqueId", UniqueIdConfigPostProcessor)
}

func UniqueIdConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet("env") {
		return false, nil
	}

	env := config.GetString("env")

	if env != "dev" && env != "test" {
		return false, nil
	}

	if err := config.Option(cfg.WithConfigSetting("unique_id.machine_id", defaultMachineId, cfg.SkipExisting)); err != nil {
		return false, fmt.Errorf("could not set unique_id.machine_id: %w", err)
	}

	if err := config.Option(cfg.WithConfigSetting("unique_id.type", defaultGeneratorType, cfg.SkipExisting)); err != nil {
		return false, fmt.Errorf("could not set unique_id.type: %w", err)
	}

	return true, nil
}
