package mon

import (
	"encoding/json"
	"fmt"
	"os"
)

func formatterGelf(timestamp string, level string, msg string, err error, data *Metadata) ([]byte, error) {
	gelf := make(Fields, 8)

	if err != nil {
		gelf["_err"] = err.Error()
	}

	for k, v := range data.Fields {
		gelf["_"+k] = v
	}

	for k, v := range data.ContextFields {
		gelf["_context_"+k] = v
	}

	gelf["version"] = "1.1"
	gelf["short_message"] = msg
	gelf["timestamp"] = timestamp
	gelf["_channel"] = data.Channel
	gelf["level"] = levels[level]
	gelf["level_name"] = level
	gelf["_pid"] = os.Getpid()

	serialized, err := json.Marshal(gelf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
