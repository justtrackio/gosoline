package metric

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Datum struct {
	Priority   int          `json:"-"`
	Timestamp  time.Time    `json:"timestamp"`
	MetricName string       `json:"metricName"`
	Dimensions Dimensions   `json:"dimensions"`
	Value      float64      `json:"value"`
	Unit       StandardUnit `json:"unit"`
	Kind       Kind         `json:"-"`
}

func (d *Datum) Id() string {
	return fmt.Sprintf("%s:%s", d.MetricName, d.DimensionKV())
}

func (d *Datum) DimensionKV() string {
	dims := make([]string, 0)

	for k, v := range d.Dimensions {
		flat := fmt.Sprintf("%s:%s", k, v)
		dims = append(dims, flat)
	}

	sort.Strings(dims)
	dimKey := strings.Join(dims, "-")

	return dimKey
}

func (d *Datum) IsValid() error {
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

type Data []*Datum
