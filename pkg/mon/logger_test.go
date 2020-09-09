package mon_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type ContextData struct {
	Version int
	Headers map[string][]string
	Nested  map[int]map[bool]*ContextData
	Data    [4]uint8
	private bool
}

func TestLogger_mergeMapStringInterface(t *testing.T) {
	logger, out := getLogger()
	err := logger.Option(mon.WithContextFieldsResolver(func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{
			"data": ContextData{
				Version: 42,
				Headers: map[string][]string{
					"Accept":   {"json", "xml", "csv"},
					"Provides": {"json"},
					"Nil":      nil,
					"Empty":    {},
				},
				Nested: map[int]map[bool]*ContextData{
					1: {
						false: nil,
						true:  &ContextData{},
					},
					4: {
						true: &ContextData{
							Version: 21,
							Headers: nil,
							Nested: map[int]map[bool]*ContextData{
								0: nil,
							},
							Data:    [4]uint8{5, 6, 7, 8},
							private: true,
						},
					},
				},
				Data:    [4]uint8{1, 2, 3, 4},
				private: true,
			},
		}
	}))
	assert.NoError(t, err)

	logger.
		WithFields(mon.Fields{
			"a field": map[string]map[string]interface{}{
				"with a": {
					"value": "of 42",
				},
			},
		}).
		WithContext(context.Background()).
		WithChannel("my channel").
		Info("my awesome log message")

	parsed := make(map[string]interface{})
	err = json.Unmarshal(out.Bytes(), &parsed)
	assert.NoError(t, err)

	assert.Equal(t, map[string]interface{}{
		"channel": "my channel",
		"context": map[string]interface{}{
			"data": map[string]interface{}{
				"Version": 42.0,
				"Headers": map[string]interface{}{
					"Accept": []interface{}{
						"json",
						"xml",
						"csv",
					},
					"Provides": []interface{}{
						"json",
					},
					"Nil":   nil,
					"Empty": []interface{}{},
				},
				"Nested": map[string]interface{}{
					"1": map[string]interface{}{
						"false": nil,
						"true": map[string]interface{}{
							"Version": 0.0,
							"Headers": map[string]interface{}{},
							"Nested":  map[string]interface{}{},
							"Data": []interface{}{
								0.0, 0.0, 0.0, 0.0,
							},
						},
					},
					"4": map[string]interface{}{
						"true": map[string]interface{}{
							"Version": 21.0,
							"Headers": map[string]interface{}{},
							"Nested": map[string]interface{}{
								"0": map[string]interface{}{},
							},
							"Data": []interface{}{
								5.0, 6.0, 7.0, 8.0,
							},
						},
					},
				},
				"Data": []interface{}{
					1.0, 2.0, 3.0, 4.0,
				},
			},
		},
		"fields": map[string]interface{}{
			"a field": map[string]interface{}{
				"with a": map[string]interface{}{
					"value": "of 42",
				},
			},
		},
		"level":      2.0,
		"level_name": "info",
		"message":    "my awesome log message",
		"timestamp":  "1984-04-04T00:00:00Z",
	}, parsed)
}

func TestLogger_WithChannel(t *testing.T) {
	gosoLog, out := getLogger()

	gosoLog.Info("msg1")

	expected := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"msg1","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")

	out.Reset()
	logger := gosoLog.WithChannel("newChannel")
	logger.Info("msg2")

	expected = `{"fields":{},"context":{},"channel": "newChannel", "level":2,"level_name":"info","message":"msg2","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")

}

func TestLogger_WithContext(t *testing.T) {
	logger, out := getLogger()
	_ = logger.Option(mon.WithContextFieldsResolver(mon.ContextLoggerFieldsResolver))

	ctx := mon.NewLoggerContext(context.Background(), mon.Fields{
		"field1": "a",
		"field2": 1,
	})

	logger.WithContext(ctx).Info("msg")

	expected := `{"fields":{},"context":{"field1":"a","field2":1},"channel": "default", "level":2,"level_name":"info","message":"msg","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_WithFields(t *testing.T) {
	logger0, out := getLogger()

	logger0.Info("test")
	expected0 := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"test","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected0, out.String(), "output should match")

	timeValue, _ := time.Parse(time.RFC3339, "2020-01-31T15:29:51+02:00")

	out.Reset()
	logger1 := logger0.WithFields(mon.Fields{
		"field1": "a",
		"field2": 1,
		"time":   timeValue,
	})
	logger1.Info("foobar")

	expected := `{"fields":{"field1":"a","field2":1,"time":"2020-01-31T15:29:51+02:00"},"context":{},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")

	out.Reset()
	logger2 := logger1.WithFields(mon.Fields{
		"field3": 0.3,
	})
	logger2.Info("msg2")

	expected = `{"fields":{"field1":"a","field2":1,"time":"2020-01-31T15:29:51+02:00","field3":0.3},"context":{},"channel": "default", "level":2,"level_name":"info","message":"msg2","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")

	out.Reset()
	logger0.Info("no fields")
	expected = `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"no fields","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_WithContext_FieldRewrite(t *testing.T) {
	logger, out := getLogger()
	_ = logger.Option(mon.WithContextFieldsResolver(mon.ContextLoggerFieldsResolver))

	ctx := mon.AppendLoggerContextField(context.Background(), mon.Fields{
		"foo": "bar",
		"faz": 1337,
	})

	ctx = mon.AppendLoggerContextField(ctx, mon.Fields{
		"foo": "foobar",
		"bar": "foo",
	})

	logger.WithContext(ctx).Info("foobar")

	expected := `{"fields":{},"context":{"faz":1337,"foo":"foobar","bar":"foo"},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_Info(t *testing.T) {
	logger, out := getLogger()

	logger.Info("bla")

	expected := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"bla","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_Infof(t *testing.T) {
	logger, out := getLogger()

	logger.Infof("this is %s formatted %v with an integer of %d", "a", "string", 10)

	expected := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"this is a formatted string with an integer of 10","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func getLogger() (mon.GosoLog, *bytes.Buffer) {
	clock := clockwork.NewFakeClock()
	out := bytes.NewBuffer([]byte{})

	logger := mon.NewLoggerWithInterfaces()

	handler, err := mon.NewIowriterLoggerHandler(clock, mon.FormatJson, out, time.RFC3339, []string{mon.Info, mon.Warn})
	if err != nil {
		panic(err)
	}

	opt := mon.WithHandler(handler)
	err = opt(logger)
	if err != nil {
		panic(err)
	}

	return logger, out
}
