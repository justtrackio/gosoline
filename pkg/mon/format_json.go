package mon

import (
	"encoding/json"
	"fmt"
)

func formatterJson(timestamp string, level string, msg string, err error, data *Metadata) ([]byte, error) {
	jsn := make(Fields, 8)

	if err != nil {
		jsn["err"] = err.Error()
	}

	jsn["channel"] = data.channel
	jsn["level"] = levels[level]
	jsn["level_name"] = level
	jsn["timestamp"] = timestamp
	jsn["message"] = msg
	jsn["fields"] = data.fields
	jsn["context"] = data.contextFields

	serialized, err := json.Marshal(jsn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
