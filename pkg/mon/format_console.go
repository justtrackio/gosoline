package mon

import (
	"fmt"
	"github.com/fatih/color"
	"strings"
)

func formatterConsole(timestamp string, level string, msg string, err error, data *Metadata) ([]byte, error) {
	fieldString := getFieldsAsString(data.Fields)
	contextString := getFieldsAsString(data.ContextFields)

	errStr := ""
	if err != nil {
		errStr = fmt.Sprintf("ERR: %s", err.Error())
	}

	level = fmt.Sprintf("%-7v", level)
	channel := fmt.Sprintf("%-7s", data.Channel)

	output := fmt.Sprintf("%s %s %s %-50s %s %s %s",
		color.YellowString(timestamp),
		color.GreenString(channel),
		color.GreenString(level),
		msg,
		color.GreenString(contextString),
		color.BlueString(fieldString),
		color.RedString(errStr),
	)

	output = strings.TrimSpace(output)
	serialized := []byte(output)

	return append(serialized, '\n'), nil
}

func getFieldsAsString(fields map[string]interface{}) string {
	fieldParts := make([]string, 0, len(fields))

	for k, v := range fields {
		fieldParts = append(fieldParts, fmt.Sprintf("%v: %v", k, v))
	}

	return strings.Join(fieldParts, ", ")
}
