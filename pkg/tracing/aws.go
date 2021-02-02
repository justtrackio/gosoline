package tracing

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/aws/aws-xray-sdk-go/xraylog"
)

func AWS(config cfg.Config, c *client.Client) {
	enabled := config.GetBool("tracing_enabled")

	if !enabled {
		return
	}

	xray.AWS(c)
}

type xrayLogger struct {
	logger mon.Logger
}

func newXrayLogger(logger mon.Logger) *xrayLogger {
	return &xrayLogger{
		logger: logger.WithChannel("tracing"),
	}
}

func (x xrayLogger) Log(level xraylog.LogLevel, msg fmt.Stringer) {
	switch level {
	case xraylog.LogLevelDebug:
		x.logger.WithFields(mon.Fields{
			"xrayLogLevel": "debug",
		}).Debug(msg)
	case xraylog.LogLevelInfo:
		x.logger.WithFields(mon.Fields{
			"xrayLogLevel": "info",
		}).Info(msg)
	case xraylog.LogLevelWarn:
		x.logger.WithFields(mon.Fields{
			"xrayLogLevel": "warn",
		}).Warn(msg)
	case xraylog.LogLevelError:
		x.logger.WithFields(mon.Fields{
			"xrayLogLevel": "error",
		}).Warn(msg.String()) // TODO we set error to warn level to prevent triggering alarm when message too long appears
	}
}
