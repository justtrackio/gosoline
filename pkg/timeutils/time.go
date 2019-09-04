package timeutils

import "time"

var defaultDateTimeFormat = "2006-01-02T15:04:05-07:00"

func WithDefaultDateTimeFormat(format string) {
	defaultDateTimeFormat = format
}

func FormatDateTime(datetime time.Time) string {
	return datetime.Format(defaultDateTimeFormat)
}

func ParseDateTime(datetime string) (time.Time, error) {
	return time.Parse(defaultDateTimeFormat, datetime)
}

func IsSameDay(a time.Time, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}
