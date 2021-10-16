package uniqueid

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/sony/sonyflake/awsutil"
)

func init() {
	// has to run after the dx post processor in gosoline/pkg/dx/unique_id.go
	cfg.AddPostProcessor(1, "gosoline.uniqueId", ConfigPostProcessor)
}

func ConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	// if there is no config, the common case is to fetch ids remotely
	if !config.IsSet("unique_id") {
		err := config.Option(cfg.WithConfigSetting(ConfigGeneratorType, GeneratorTypeSrv))
		if err != nil {
			return false, fmt.Errorf("could not set generator type: %w", err)
		}

		return true, nil
	}

	// for sonyflake generators, the machine id is derived from the ip
	generatorType := config.GetString(ConfigGeneratorType)
	if generatorType != GeneratorTypeSonyFlake {
		return false, nil
	}

	machineIdCfg := config.GetInt(ConfigMachineId, 0)
	machineId := uint16(machineIdCfg)

	if machineId > 0 {
		return false, nil
	}

	machineId, err := awsutil.AmazonEC2MachineID()
	if err != nil {
		return false, fmt.Errorf("could not read ec2 machine id: %w", err)
	}

	if err := config.Option(cfg.WithConfigSetting(ConfigMachineId, machineId)); err != nil {
		return false, fmt.Errorf("could not set unique_id.machine_id: %w", err)
	}

	return true, nil
}
