package mon

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"sort"
	"strings"
	"time"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	UnitCount        = cloudwatch.StandardUnitCount
	UnitCountAverage = "UnitCountAverage"
	UnitSeconds      = cloudwatch.StandardUnitSeconds
	UnitMilliseconds = cloudwatch.StandardUnitMilliseconds
)

type MetricDimensions map[string]string

type MetricDatum struct {
	Priority   int              `json:"-"`
	Timestamp  time.Time        `json:"timestamp"`
	MetricName string           `json:"metricName"`
	Dimensions MetricDimensions `json:"dimensions"`
	Value      float64          `json:"value"`
	Unit       string           `json:"unit"`
}

func (d *MetricDatum) Id() string {
	return fmt.Sprintf("%s:%s", d.MetricName, d.DimensionKey())
}

func (d *MetricDatum) DimensionKey() string {
	dims := make([]string, 0)

	for k, v := range d.Dimensions {
		flat := fmt.Sprintf("%s:%s", k, v)
		dims = append(dims, flat)
	}

	sort.Strings(dims)
	dimKey := strings.Join(dims, "-")

	return dimKey
}

func (d *MetricDatum) IsValid() error {
	if d.MetricName == "" {
		return fmt.Errorf("missing metric name")
	}

	if d.Priority == 0 {
		return fmt.Errorf("metric %s has no priority", d.MetricName)
	}

	if d.Unit == "" {
		return fmt.Errorf("metric %s has no unit", d.MetricName)
	}

	return nil
}

type MetricData []*MetricDatum

//go:generate mockery -name MetricWriter
type MetricWriter interface {
	GetPriority() int
	Write(batch MetricData)
	WriteOne(data *MetricDatum)
}
