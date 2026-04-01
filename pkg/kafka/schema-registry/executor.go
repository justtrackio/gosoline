package schema_registry

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/log"
)

// NewExecutor builds the schema registry executor used to retry transient HTTP startup failures.
func NewExecutor(config cfg.Config, logger log.Logger, connectionName string, settings connection.Settings) (exec.Executor, error) {
	backoffType := fmt.Sprintf("kafka.connection.%s.schema_registry.retry", connectionName)

	backoffSettings, err := exec.ReadBackoffSettings(config, backoffType)
	if err != nil {
		return nil, fmt.Errorf("can not read backoff settings: %w", err)
	}

	resource := &exec.ExecutableResource{
		Type: "schema-registry",
		Name: settings.SchemaRegistryAddress,
	}

	return exec.NewExecutor(logger, resource, &backoffSettings, []exec.ErrorChecker{
		exec.CheckConnectionError,
		exec.CheckTimeoutError,
		exec.CheckClientAwaitHeaderTimeoutError,
		exec.CheckTlsHandshakeTimeoutError,
		exec.CheckHttp2ClientConnectionForceClosedError,
	}), nil
}
