package mon

import (
	"strconv"
	"time"
)

const (
	UnixNano    = "unix_n"
	UnixSeconds = "unix_s"
)

func FormatTime(time time.Time, format string) string {
	switch format {
	case UnixNano:
		return strconv.FormatInt(time.UnixNano(), 10)
	case UnixSeconds:
		return strconv.FormatInt(time.Unix(), 10)
	default:
		return time.Format(format)
	}
}
