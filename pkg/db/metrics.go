package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	metricNamespace             = "db.client"
	metricNameDbConnectionCount = "connection.count"
	metricNameDbConnections     = "connections"

	// dimensionConnectionState is the semantic-convention attribute for the state a pooled database
	// connection is in.
	dimensionConnectionState = "db.client.connection.state"

	connectionStateIdle = "idle"
	connectionStateUsed = "used"
)

type metricDriver struct {
	driver.Driver

	metricWriter metric.Writer
}

func newMetricDriver(driver driver.Driver) string {
	mw := metric.NewWriter(metricNamespace)

	id := uuid.New().NewV4()
	md := &metricDriver{
		Driver:       driver,
		metricWriter: mw,
	}

	sql.Register(id, md)

	return id
}

func (m *metricDriver) Open(dsn string) (driver.Conn, error) {
	m.metricWriter.WriteOne(context.Background(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameDbConnections,
		Unit:       metric.UnitCount,
		Value:      1.0,
		Kind:       metric.KindCounter.Build(),
	})

	return m.Driver.Open(dsn)
}

func publishConnectionMetrics(ctx context.Context, conn *sqlx.DB) {
	publishConnectionMetricsWithInterfaces(ctx, conn, clock.Provider, metric.NewWriter(metricNamespace))
}

func publishConnectionMetricsWithInterfaces(ctx context.Context, conn *sqlx.DB, clk clock.Clock, writer metric.Writer) {
	go func() {
		ticker := clk.NewTicker(time.Minute)
		defer ticker.Stop()

		write := func() {
			stats := conn.Stats()

			// the total number of open connections is the sum of the states, so it is not emitted
			writer.Write(ctx, metric.Data{
				connectionCountDatum(connectionStateUsed, stats.InUse),
				connectionCountDatum(connectionStateIdle, stats.Idle),
			})
		}

		write()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.Chan():
				write()
			}
		}
	}()
}

func connectionCountDatum(state string, count int) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameDbConnectionCount,
		Dimensions: map[string]string{
			dimensionConnectionState: state,
		},
		Unit:  metric.UnitCountAverage,
		Value: float64(count),
		Kind:  metric.KindGauge.Build(),
	}
}
