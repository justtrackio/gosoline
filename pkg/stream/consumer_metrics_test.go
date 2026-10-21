package stream_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type ignorableUnknownModelError struct{}

func (ignorableUnknownModelError) Error() string {
	return "unknown model"
}

func (ignorableUnknownModelError) IsIgnorableWithSettings(settings stream.IgnoreOnGetModelErrorSettings) bool {
	return settings.UnknownModel
}

func findPrometheusCounter(registry *prometheus.Registry, familyName string, dimensions map[string]string) (value float64, found bool, err error) {
	families, err := registry.Gather()
	if err != nil {
		return 0, false, err
	}

	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}

		if family.GetType() != dto.MetricType_COUNTER {
			return 0, false, fmt.Errorf("metric family %q is not a counter", familyName)
		}

		for _, sample := range family.GetMetric() {
			labels := make(map[string]string, len(sample.GetLabel()))
			for _, label := range sample.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			if equalMetricLabels(labels, dimensions) {
				return sample.GetCounter().GetValue(), true, nil
			}
		}
	}

	return 0, false, nil
}

func requirePrometheusCounter(t testing.TB, registry *prometheus.Registry, familyName string, dimensions map[string]string, value float64) {
	t.Helper()

	got, found, err := findPrometheusCounter(registry, familyName, dimensions)
	require.NoError(t, err)
	require.True(t, found, "counter %q with dimensions %v was not exported", familyName, dimensions)
	require.Equal(t, value, got)
}

func equalMetricLabels(actual, expected map[string]string) bool {
	if len(actual) != len(expected) {
		return false
	}

	for key, value := range expected {
		if actual[key] != value {
			return false
		}
	}

	return true
}
