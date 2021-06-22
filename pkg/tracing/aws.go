package tracing

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
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
	logger log.Logger
}

func newXrayLogger(logger log.Logger) *xrayLogger {
	return &xrayLogger{
		logger: logger.WithChannel("tracing"),
	}
}

func (x xrayLogger) Log(level xraylog.LogLevel, msg fmt.Stringer) {
	switch level {
	case xraylog.LogLevelDebug:
		x.logger.WithFields(log.Fields{
			"xrayLogLevel": "debug",
		}).Debug(msg.String())
	case xraylog.LogLevelInfo:
		x.logger.WithFields(log.Fields{
			"xrayLogLevel": "info",
		}).Info(msg.String())
	case xraylog.LogLevelWarn:
		x.logger.WithFields(log.Fields{
			"xrayLogLevel": "warn",
		}).Warn(msg.String())
	case xraylog.LogLevelError:
		x.logger.WithFields(log.Fields{
			"xrayLogLevel": "error",
		}).Warn(msg.String()) // TODO we set error to warn level to prevent triggering alarm when message too long appears
	}
}
