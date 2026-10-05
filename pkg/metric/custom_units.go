package metric

import (
	"fmt"
	"math"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

const (
	UnitCountAverage        = types.StandardUnit("UnitCountAverage")
	UnitCountMaximum        = types.StandardUnit("UnitCountMaximum")
	UnitCountMinimum        = types.StandardUnit("UnitCountMinimum")
	UnitSecondsAverage      = types.StandardUnit("UnitSecondsAverage")
	UnitSecondsMaximum      = types.StandardUnit("UnitSecondsMaximum")
	UnitSecondsMinimum      = types.StandardUnit("UnitSecondsMinimum")
	UnitMillisecondsAverage = types.StandardUnit("UnitMillisecondsAverage")
	UnitMillisecondsMaximum = types.StandardUnit("UnitMillisecondsMaximum")
	UnitMillisecondsMinimum = types.StandardUnit("UnitMillisecondsMinimum")
)

var customUnits map[types.StandardUnit]unitDefinition

type unitDefinition struct {
	Unit    types.StandardUnit
	Reducer func(xs []float64) float64
}

func init() {
	customUnits = make(map[types.StandardUnit]unitDefinition)

	RegisterCustomUnit(UnitCountAverage, UnitCount, average)
	RegisterCustomUnit(UnitCountMaximum, UnitCount, maximum)
	RegisterCustomUnit(UnitCountMinimum, UnitCount, minimum)
	RegisterCustomUnit(UnitMillisecondsAverage, UnitMilliseconds, average)
	RegisterCustomUnit(UnitMillisecondsMaximum, UnitMilliseconds, maximum)
	RegisterCustomUnit(UnitMillisecondsMinimum, UnitMilliseconds, minimum)
	RegisterCustomUnit(UnitSecondsAverage, UnitSeconds, average)
	RegisterCustomUnit(UnitSecondsMaximum, UnitSeconds, maximum)
	RegisterCustomUnit(UnitSecondsMinimum, UnitSeconds, minimum)
}

func RegisterCustomUnit(unit types.StandardUnit, standardUnit types.StandardUnit, reducer func(xs []float64) float64) {
	if _, ok := customUnits[unit]; ok {
		panic(fmt.Errorf("can not register %s as %s, a custom unit with name %s was already registered", unit, standardUnit, unit))
	}

	customUnits[unit] = unitDefinition{
		Unit:    standardUnit,
		Reducer: reducer,
	}
}

func resolveCustomUnit(unit types.StandardUnit, values []float64) (resolvedUnit types.StandardUnit, resolvedValue float64) {
	if customMetric, ok := customUnits[unit]; ok {
		return customMetric.Unit, customMetric.Reducer(values)
	}

	return unit, sum(values)
}

// resolveBaseUnit maps a custom aggregation unit to the unit it actually measures. A custom unit
// expresses how batched values are reduced rather than what is measured, so every writer resolves it
// before rendering the unit. A unit that is not a custom aggregation unit is returned unchanged.
func resolveBaseUnit(unit types.StandardUnit) types.StandardUnit {
	if customMetric, ok := customUnits[unit]; ok {
		return customMetric.Unit
	}

	return unit
}

func average(xs []float64) float64 {
	return sum(xs) / float64(len(xs))
}

func sum(xs []float64) float64 {
	total := 0.0

	for _, v := range xs {
		total += v
	}

	return total
}

func maximum(xs []float64) float64 {
	result := math.Inf(-1)

	for _, v := range xs {
		result = math.Max(result, v)
	}

	return result
}

func minimum(xs []float64) float64 {
	result := math.Inf(1)

	for _, v := range xs {
		result = math.Min(result, v)
	}

	return result
}
