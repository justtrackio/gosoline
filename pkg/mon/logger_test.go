package mon_test

import (
	"bytes"
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

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

	out.Reset()
	logger1 := logger0.WithFields(mon.Fields{
		"field1": "a",
		"field2": 1,
	})
	logger1.Info("foobar")

	expected := `{"fields":{"field1":"a","field2":1},"context":{},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":"1984-04-04T00:00:00Z"}`
	assert.JSONEq(t, expected, out.String(), "output should match")

	out.Reset()
	logger2 := logger1.WithFields(mon.Fields{
		"field3": 0.3,
	})
	logger2.Info("msg2")

	expected = `{"fields":{"field1":"a","field2":1, "field3":0.3},"context":{},"channel": "default", "level":2,"level_name":"info","message":"msg2","timestamp":"1984-04-04T00:00:00Z"}`
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

	client := mon.NewLoggerWithInterfaces(clock, out)
	err := client.Option(mon.WithFormat(mon.FormatJson), mon.WithTimestampFormat(time.RFC3339))

	if err != nil {
		panic(err)
	}

	return client, out
}
