package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
)

func formatterJson(clock clockwork.Clock, channel string, level string, msg string, logErr error, fields Fields, contextFields ContextFields) ([]byte, error) {
	data := make(Fields, 8)

	if logErr != nil {
		data["err"] = logErr.Error()
	}

	data["channel"] = channel
	data["level"] = levels[level]
	data["level_name"] = level
	data["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	data["message"] = msg
	data["fields"] = fields
	data["context"] = contextFields

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
