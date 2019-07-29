package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
	"os"
)

func formatterGelf(clock clockwork.Clock, channel string, level string, msg string, logErr error, fields Fields, contextFields ContextFields) ([]byte, error) {
	data := make(Fields, 8)

	if logErr != nil {
		data["_err"] = logErr.Error()
	}

	for k, v := range fields {
		data["_"+k] = v
	}

	for k, v := range contextFields {
		data["_context_"+k] = v
	}

	data["version"] = "1.1"
	data["short_message"] = msg
	data["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	data["_channel"] = channel
	data["level"] = levels[level]
	data["level_name"] = level
	data["_pid"] = os.Getpid()

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
