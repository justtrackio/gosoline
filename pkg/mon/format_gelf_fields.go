package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
	"os"
)

func formatterGelfFields(clock clockwork.Clock, channel string, level string, msg string, logErr error, fields Fields) ([]byte, error) {
	data := make(Fields, 8)

	if logErr != nil {
		fields["err"] = logErr.Error()
	}

	jsonFields, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}
	data["_fields"] = string(jsonFields)

	data["version"] = "1.1"
	data["short_message"] = msg
	data["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	data["channel"] = channel
	data["level"] = levels[level]
	data["level_name"] = level
	data["pid"] = os.Getpid()

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal log message to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
