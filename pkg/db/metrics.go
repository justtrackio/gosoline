package db

import (
	"database/sql"
	"database/sql/driver"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/jmoiron/sqlx"
	"github.com/twinj/uuid"
	"time"
)

const (
	metricNameDbConnectionCount = "DbConnectionCount"
)

type metricDriver struct {
	driver.Driver

	metricWriter metric.Writer
}

func newMetricDriver(driver driver.Driver) string {
	mw := metric.NewDaemonWriter()

	id := uuid.NewV4().String()
	md := &metricDriver{
		Driver:       driver,
		metricWriter: mw,
	}

	sql.Register(id, md)

	return id
}

func (m *metricDriver) Open(dsn string) (driver.Conn, error) {
	m.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameDbConnectionCount,
		Dimensions: map[string]string{
			"Type": "new",
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})

	return m.Driver.Open(dsn)
}

func publishConnectionMetrics(conn *sqlx.DB) {
	output := metric.NewDaemonWriter()

	go func() {
		for {
			stats := conn.Stats()

			output.Write(metric.Data{
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "open",
					},
					Unit:  metric.UnitCountAverage,
					Value: float64(stats.OpenConnections),
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "inUse",
					},
					Unit:  metric.UnitCountAverage,
					Value: float64(stats.InUse),
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "idle",
					},
					Unit:  metric.UnitCountAverage,
					Value: float64(stats.Idle),
				},
			})

			time.Sleep(time.Minute)
		}
	}()
}
