package tracing

import (
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/aws/aws-xray-sdk-go/xraylog"
	"github.com/justtrackio/gosoline/pkg/log"
)

var globalXRayLogger = &xrayLogger{
	fallback: xraylog.NewDefaultLogger(os.Stdout, xraylog.LogLevelInfo),
}

func init() {
	// SetLogger is not thread safe, so set it upon startup to our wrapper. Our wrapper is safe and falls back to the
	// default if we don't have our logger initialized yet.
	xray.SetLogger(globalXRayLogger)
}

type xrayLogger struct {
	lck      sync.RWMutex
	logger   log.Logger
	fallback xraylog.Logger
}

func setGlobalXRayLogger(logger log.Logger) {
	globalXRayLogger.lck.Lock()
	defer globalXRayLogger.lck.Unlock()

	globalXRayLogger.logger = logger.WithChannel("tracing")
}

func (x *xrayLogger) Log(level xraylog.LogLevel, msg fmt.Stringer) {
	globalXRayLogger.lck.RLock()
	defer globalXRayLogger.lck.RUnlock()

	if x.logger == nil {
		x.fallback.Log(level, msg)
		return
	}

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
