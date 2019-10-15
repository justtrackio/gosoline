package db

import (
	"database/sql"
	"database/sql/driver"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"time"
)

func init() {
	sql.Register("metricWrapper", newMetricDriver())
}

const (
	metricNameDbConnectionCount = "DbConnectionCount"
)

type metricDriver struct {
	metricWriter mon.MetricWriter
}

func newMetricDriver() *metricDriver {
	mw := mon.NewMetricDaemonWriter()

	return &metricDriver{
		metricWriter: mw,
	}
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

	return mysql.MySQLDriver{}.Open(dsn)
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
