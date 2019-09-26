package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
	"os"
)

func formatterGelfFields(clock clockwork.Clock, level string, msg string, err error, data *Metadata) ([]byte, error) {
	gelf := make(Fields, 8)

	if err != nil {
		gelf["_err"] = err.Error()
	}

	jsonFields, err := json.Marshal(data.fields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}
	gelf["_fields"] = string(jsonFields)

	contextFields, err := json.Marshal(data.fields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}
	gelf["_context"] = string(contextFields)

	gelf["version"] = "1.1"
	gelf["short_message"] = msg
	gelf["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	gelf["channel"] = data.channel
	gelf["level"] = levels[level]
	gelf["level_name"] = level
	gelf["_pid"] = os.Getpid()

	serialized, err := json.Marshal(gelf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log message to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
