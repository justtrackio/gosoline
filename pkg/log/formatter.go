package log

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
)

type Formatter func(timestamp string, level int, format string, args []interface{}, err error, data Data) ([]byte, error)

var formatters = map[string]Formatter{
	"console": FormatterConsole,
	"json":    FormatterJson,
}

func FormatterConsole(timestamp string, level int, format string, args []interface{}, err error, data Data) ([]byte, error) {
	fieldString := getFieldsAsString(data.Fields)
	contextString := getFieldsAsString(data.ContextFields)

	levelStr := fmt.Sprintf("%-7s", LevelName(level))
	channel := fmt.Sprintf("%-7s", data.Channel)
	msg := fmt.Sprintf(format, args...)

	if err != nil {
		msg = color.RedString(err.Error())
	}

	output := fmt.Sprintf("%s %s %s %-50s %s %s",
		color.YellowString(timestamp),
		color.GreenString(channel),
		color.GreenString(levelStr),
		msg,
		color.GreenString(contextString),
		color.BlueString(fieldString),
	)

	output = strings.TrimSpace(output)
	serialized := []byte(output)

	return append(serialized, '\n'), nil
}

type formatterJsonStruct struct {
	Error     string                 `json:"err,omitempty"`
	Channel   string                 `json:"channel"`
	Level     int                    `json:"level"`
	LevelName string                 `json:"level_name"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Context   map[string]interface{} `json:"context"`
}

func FormatterJson(timestamp string, level int, format string, args []interface{}, err error, data Data) ([]byte, error) {
	msg := fmt.Sprintf(format, args...)

	jsn := &formatterJsonStruct{
		Channel:   data.Channel,
		Level:     level,
		LevelName: LevelName(level),
		Timestamp: timestamp,
		Message:   msg,
		Fields:    data.Fields,
		Context:   data.ContextFields,
	}

	if err != nil {
		jsn.Error = err.Error()
	}

	serialized, err := json.Marshal(jsn)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}

	return append(serialized, '\n'), nil
}

func getFieldsAsString(fields map[string]interface{}) string {
	keys := make([]string, 0, len(fields))
	fieldParts := make([]string, 0, len(fields))

	for k := range fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		fieldParts = append(fieldParts, fmt.Sprintf("%v: %v", k, fields[k]))
	}

	return strings.Join(fieldParts, ", ")
}
