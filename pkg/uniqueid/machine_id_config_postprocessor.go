package uniqueid

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/sony/sonyflake/awsutil"
)

type MachineIdSettings struct {
	MachineId uint16 `cfg:"machine_id"`
}

func init() {
	// has to run after the dx post processor in gosoline/pkg/dx/unique_id.go
	cfg.AddPostProcessor(1, "gosoline.uniqueId", ConfigPostProcessor)
}

func ConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet("unique_id") {
		return false, nil
	}

	settings := MachineIdSettings{}
	config.UnmarshalKey("unique_id", &settings)

	if settings.MachineId > 0 {
		return false, nil
	}

	machineId, err := awsutil.AmazonEC2MachineID()
	if err != nil {
		return false, fmt.Errorf("could not read ec2 machine id: %w", err)
	}

	if err := config.Option(cfg.WithConfigSetting("unique_id.machine_id", machineId)); err != nil {
		return false, fmt.Errorf("could not set unique_id.machine_id: %w", err)
	}

	return true, nil
}
