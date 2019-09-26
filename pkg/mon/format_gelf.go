package mon

import (
	"encoding/json"
	"fmt"
	"github.com/jonboulle/clockwork"
	"os"
)

func formatterGelf(clock clockwork.Clock, level string, msg string, err error, data *Metadata) ([]byte, error) {
	gelf := make(Fields, 8)

	if err != nil {
		gelf["_err"] = err.Error()
	}

	for k, v := range data.fields {
		gelf["_"+k] = v
	}

	for k, v := range data.contextFields {
		gelf["_context_"+k] = v
	}

	gelf["version"] = "1.1"
	gelf["short_message"] = msg
	gelf["timestamp"] = round((float64(clock.Now().UnixNano())/float64(1000000))/float64(1000), 4)
	gelf["_channel"] = data.channel
	gelf["level"] = levels[level]
	gelf["level_name"] = level
	gelf["_pid"] = os.Getpid()

	serialized, err := json.Marshal(gelf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return append(serialized, '\n'), nil
}
