# Log package

The gosoline logger is based upon a simple interface that uses handlers internally to enable fully customizable log handling. 

[embedmd]:# (../../pkg/log/logger.go /type Logger interface {/ /\n}/)
```go
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})

	WithChannel(channel string) Logger
	WithContext(ctx context.Context) Logger
	WithFields(Fields) Logger
}
```

## Manual create and customize
To get a logger with no handlers and a real time clock you can call:
```golang
logger := log.NewLogger()
```

If you want to provide a clock and have some handlers already, call:
```golang
logger := log.NewLoggerWithInterfaces(myClock, []log.Handler{handler1, handler2})
```

Both will provide you with an extended interface including the `Option(opt ...Option) error` function to change the behaviour of the logger.

##### `WithContextFieldsResolver`
Adds a context fields resolver to the logger

##### `WithFields`
Adds global fields to the logger, which will be set on every log message

##### `WithHandlers`
Adds additional handlers to the logger

## Create from config
Most of the time the logger will be created and setup based on the configuration of your application. The default logger configuration is:

```yaml
log:
    level: info
    handlers:
        main:
            type: iowriter
            channels: []
            formatter: console
            level: info
            timestamp_format: 15:04:05.000
            writer: stdout
```

| setting             | description                                                    | default                                                                            |
|---------------------|----------------------------------------------------------------|------------------------------------------------------------------------------------|
| log.level           | default level for all handlers without an explicit level value | info                                                                               |
| log.handlers        | a map of handlers that will be called for every log message   | every logger gets a 'main' handler by default if there is no other handler defined |
| log.handlers.X.type | defines the type of the handler                                | -                                                                                  |

## Handlers
Handlers have to implement the following interface:

[embedmd]:# (../../pkg/log/handler.go /type Handler interface {/ /\n}/)
```go
type Handler interface {
	Channels() []string
	Level() int
	Log(timestamp time.Time, level int, msg string, args []interface{}, err error, data Data) error
}
```

`Channels() []string` and `Level() int` are called on every log action to check if the handler should be applied. `Log` does the actual logging afterwards.

### Implementing a handler and make it available via config

[embedmd]:# (custom_handler.go)
```go
package main

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type MyCustomHandlerSettings struct {
	Channel string `cfg:"channel"`
}

func MyCustomHandlerFactory(config cfg.Config, name string) (log.Handler, error) {
	settings := &MyCustomHandlerSettings{}
	log.UnmarshalHandlerSettingsFromConfig(config, name, settings)

	return &MyCustomHandler{
		channel: settings.Channel,
	}, nil
}

type MyCustomHandler struct {
	channel string
}

func (h *MyCustomHandler) Channels() []string {
	return []string{h.channel}
}

func (h *MyCustomHandler) Level() int {
	return log.PriorityInfo
}

func (h *MyCustomHandler) Log(timestamp time.Time, level int, msg string, args []interface{}, err error, data log.Data) error {
	fmt.Printf("%s happenend at %s", msg, timestamp.Format(time.RFC822))
	return nil
}

func main() {
	log.AddHandlerFactory("my-custom-handler", MyCustomHandlerFactory)
}
```

The corresponding config will look like:
```yaml
log:
  handlers:
    main:
      type: my-custom-handler
      channel: important
```

### Build-in handlers
Gosoline has a couple of built-in handlers, which are ready to use out of the box:

#### iowriter
Multitool, which is able to write logs to everything which implements the `io.Writer` interface. Config options are:

| Setting          | Description                                                        | Default      |
|------------------|--------------------------------------------------------------------|--------------|
| level            | Levels of this and higher priority will get logged                 | info         |
| channels         | Messages logged into these channels will be handled                | []           |
| formatter        | Which format should be used by this handler                        | console      |
| timestamp_format | A golang time format string to control the format of the timestamp | 15:04:05.000 |
| writer           | Which io.writer implementation to use                              | stdout       |

##### log to stdout
```yaml
log:
  handlers:
    main:
      type: iowriter
      level: info
      channels: []
      formatter: console
      timestamp_format: 15:04:05.000
      writer: stdout
```
##### log to a file
```yaml
log:
  handlers:
    main:
      type: iowriter
      level: info
      channels: *
      formatter: console
      timestamp_format: 15:04:05.000
      writer: file
      path: logs.log
```

#### metric
No configuration needed. Writes a metric data point for every warn and error log.

#### sentry
No configuration needed. Publishes every logged error to sentry.

## Usage

[embedmd]:# (usage.go /func Usage/ /\n}/)
```go
func Usage() {
	ctx := context.Background()
	handler := log.NewHandlerIoWriter(log.LevelDebug, []string{}, log.FormatterConsole, "15:04:05.000", os.Stdout)
	logger := log.NewLoggerWithInterfaces(clock.NewRealClock(), []log.Handler{handler})

	if err := logger.Option(log.WithContextFieldsResolver(log.ContextLoggerFieldsResolver)); err != nil {
		panic(err)
	}

	logger.Info("log a number %d", 4)
	logger.WithChannel("strings").Warn("a dangerous string appeared: %s", "foobar")

	loggerWithFields := logger.WithFields(log.Fields{
		"b": true,
	})
	loggerWithFields.Debug("just some debug line")
	loggerWithFields.Error("it happens: %w", fmt.Errorf("should not happen"))

	ctx = log.AppendLoggerContextField(ctx, map[string]interface{}{
		"id": 1337,
	})

	contextAwareLogger := logger.WithContext(ctx)
	contextAwareLogger.Info("some info")
}
```

results in
```
14:03:14.631 main    info    log a number 4                                      
14:03:14.631 strings warn    a dangerous string appeared: foobar                 
14:03:14.631 main    debug   just some debug line                                b: true
14:03:14.631 main    error   it happens: should not happen                       b: true
14:03:14.631 main    info    some info                                          id: 1337 

```
