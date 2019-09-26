package mon

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/jonboulle/clockwork"
	"strings"
)

func formatterConsole(clock clockwork.Clock, level string, msg string, err error, data *Metadata) ([]byte, error) {
	fieldString := getFieldsAsString(data.fields)
	contextString := getFieldsAsString(data.contextFields)

	now := clock.Now().Format("15:04:05.999999")

	errStr := ""
	if err != nil {
		errStr = fmt.Sprintf("ERR: %s", err.Error())
	}

	now = fmt.Sprintf("%-15v", now)
	level = fmt.Sprintf("%-7v", level)
	channel := fmt.Sprintf("%-7s", data.channel)

	output := fmt.Sprintf("%s %s %s %-50s %s %s %s",
		color.YellowString(now),
		color.GreenString(channel),
		color.GreenString(level),
		msg,
		color.GreenString(contextString),
		color.BlueString(fieldString),
		color.RedString(errStr),
	)
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
