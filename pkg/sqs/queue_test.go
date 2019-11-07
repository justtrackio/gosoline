package sqs_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/mon"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/timeutils"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type connResetServer struct {
	ln net.Listener
}

func (c *connResetServer) open(t *testing.T) {
	ln, err := net.Listen("tcp", ":9999")
	assert.NoError(t, err)

	c.ln = ln
}

func (c *connResetServer) handleOnce(t *testing.T) {
	conn, err := c.ln.Accept()
	assert.NoError(t, err)

	netConn := conn.(*net.TCPConn)
	// setting linger to 0 and then closing the connection causes the connection
	// to be closed by sending a RST packet instead of FIN - we see this as
	// "Connection reset by peer" or similar in our application
	err = netConn.SetLinger(0)
	assert.NoError(t, err)

	err = conn.Close()
	assert.NoError(t, err)
}

func (c *connResetServer) close(t *testing.T) {
	err := c.ln.Close()
	assert.NoError(t, err)

	c.ln = nil
}

func TestQueue_ReceiveConnectionReset(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	config := new(mocks.Config)
	metric := new(monMocks.MetricWriter)

	config.On("GetString", "aws_sqs_endpoint").Return("http://localhost:9999")
	config.On("GetInt", "aws_sdk_retries").Return(1)

	metric.On("WriteOne", &mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Time{},
		MetricName: sqs.MetricNameQueueErrorCount,
		Dimensions: map[string]string{
			"Queue": "test-queue",
		},
		Value: 1.0,
		Unit:  mon.UnitCount,
	})

	timeutils.SetGlobalTimeProvider(timeutils.ConstantTimeProvider(time.Time{}))
	defer timeutils.SetGlobalTimeProvider(nil)

	crs := &connResetServer{}
	crs.open(t)
	defer crs.close(t)

	go crs.handleOnce(t)

	q := sqs.NewWithInterfaces(logger, sqs.GetClient(config, logger), &sqs.Properties{
		Name: "test-queue",
		Url:  "http://localhost:9999/queue/test-queue",
		Arn:  "arn:aws:sqs:eu-central-1:123456789012:test-queue",
	}, metric)
	messages, err := q.Receive(context.TODO(), 10)

	assert.NoError(t, err)
	assert.Nil(t, messages)

	config.AssertExpectations(t)
	metric.AssertExpectations(t)
}
