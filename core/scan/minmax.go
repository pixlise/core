package scan

import "math"

type MinMax struct {
	Min *float64
	Max *float64
}

func (m *MinMax) Expand(v float64) {
	m.ExpandMin(v)
	m.ExpandMax(v)
}

func (m *MinMax) ExpandMin(v float64) bool {
	if !math.IsInf(float64(v), 0) && !math.IsNaN(float64(v)) && (m.Min == nil || v < *m.Min) {
		m.Min = &v
		return true
	}
	return false
}

func (m *MinMax) ExpandMax(v float64) bool {
	if !math.IsInf(float64(v), 0) && !math.IsNaN(float64(v)) && (m.Max == nil || v > *m.Max) {
		m.Max = &v
		return true
	}
	return false
}

func (m *MinMax) GetRange() float64 {
	if m.Max == nil || m.Min == nil {
		return 0
	}
	return *m.Max - *m.Min
}
