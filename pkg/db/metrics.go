package db

import (
	"database/sql"
	"database/sql/driver"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jmoiron/sqlx"
	"github.com/twinj/uuid"
	"time"
)

const (
	metricNameDbConnectionCount = "DbConnectionCount"
)

type metricDriver struct {
	driver.Driver

	metricWriter mon.MetricWriter
}

func newMetricDriver(driver driver.Driver) string {
	mw := mon.NewMetricDaemonWriter()

	id := uuid.NewV4().String()
	md := &metricDriver{
		Driver:       driver,
		metricWriter: mw,
	}

	sql.Register(id, md)

	return id
}

func (m *metricDriver) Open(dsn string) (driver.Conn, error) {
	m.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		MetricName: metricNameDbConnectionCount,
		Dimensions: map[string]string{
			"Type": "new",
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})

	return m.Driver.Open(dsn)
}

func publishConnectionMetrics(conn *sqlx.DB) {
	output := mon.NewMetricDaemonWriter()

	go func() {
		for {
			stats := conn.Stats()

			output.Write(mon.MetricData{
				&mon.MetricDatum{
					Priority:   mon.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "open",
					},
					Unit:  mon.UnitCountAverage,
					Value: float64(stats.OpenConnections),
				},
				&mon.MetricDatum{
					Priority:   mon.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "inUse",
					},
					Unit:  mon.UnitCountAverage,
					Value: float64(stats.InUse),
				},
				&mon.MetricDatum{
					Priority:   mon.PriorityHigh,
					MetricName: metricNameDbConnectionCount,
					Dimensions: map[string]string{
						"Type": "idle",
					},
					Unit:  mon.UnitCountAverage,
					Value: float64(stats.Idle),
				},
			})

			time.Sleep(time.Minute)
		}
	}()
}
