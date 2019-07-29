package mon_test

import (
	"bytes"
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClient_WithFields(t *testing.T) {
	logger, out := getLogger()

	logger.WithFields(mon.Fields{
		"foo": "bar",
		"faz": 1337,
	}).Info("foobar")

	expected := `{"fields":{"faz":1337,"foo":"bar"},"context":{},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_WithContext(t *testing.T) {
	logger, out := getLogger()

	ctx := mon.AppendLoggerContextField(context.Background(), mon.ContextFields{
		"foo": "bar",
		"faz": 1337,
	})

	logger.WithContext(ctx).Info("foobar")

	expected := `{"fields":{"trace_id":""},"context":{"faz":1337,"foo":"bar"},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_WithContext_Empty(t *testing.T) {
	logger, out := getLogger()

	ctx := context.Background()

	logger.WithContext(ctx).Info("foobar")

	expected := `{"fields":{"trace_id":""},"context":{},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_WithContext_Rewrite(t *testing.T) {
	logger, out := getLogger()

	ctx := mon.AppendLoggerContextField(context.Background(), mon.ContextFields{
		"foo": "bar",
		"faz": 1337,
	})

	ctx = mon.AppendLoggerContextField(ctx, mon.ContextFields{
		"foo": "foobar",
		"bar": "foo",
	})

	logger.WithContext(ctx).Info("foobar")

	expected := `{"fields":{"trace_id":""},"context":{"faz":1337,"foo":"foobar","bar":"foo"},"channel": "default", "level":2,"level_name":"info","message":"foobar","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_Info(t *testing.T) {
	logger, out := getLogger()

	logger.Info("bla")

	expected := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"bla","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func TestClient_Infof(t *testing.T) {
	logger, out := getLogger()

	logger.Infof("this is %s formatted %v with an integer of %d", "a", "string", 10)

	expected := `{"fields":{},"context":{},"channel": "default", "level":2,"level_name":"info","message":"this is a formatted string with an integer of 10","timestamp":449884800}`
	assert.JSONEq(t, expected, out.String(), "output should match")
}

func getLogger() (mon.Logger, *bytes.Buffer) {
	clock := clockwork.NewFakeClock()
	out := bytes.NewBuffer([]byte{})

	client := mon.NewLoggerWithInterfaces(clock, out, mon.Trace, mon.FormatJson, mon.Tags{}, mon.ConfigValues{})

	return client, out
}
