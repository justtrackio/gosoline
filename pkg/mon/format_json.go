package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
)

func formatterJson(clock clockwork.Clock, level string, msg string, err error, data *Metadata) ([]byte, error) {
	jsn := make(Fields, 8)

	if err != nil {
		jsn["err"] = err.Error()
	}

	jsn["channel"] = data.channel
	jsn["level"] = levels[level]
	jsn["level_name"] = level
	jsn["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	jsn["message"] = msg
	jsn["fields"] = data.fields
	jsn["context"] = data.contextFields

	serialized, err := json.Marshal(jsn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
