// Package metric provides defining of metrics used in monitoring systems.
package metric

import (
	"strconv"
)

// Metric represents a single metric used in monitoring systems.
//
// This struct encapsulates the properties of a metric, including its unique identifier (name),
// type, and value.
type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

// ValueAsString returns the string representation of the metric's value based on its type.
//
// Returns:
//   - The string representation of the metric's value, formatted according to its type.
func (m *Metric) ValueAsString() string {
	if m.MType == "gauge" {
		return strconv.FormatFloat(*m.Value, 'g', -1, 64)
	}
	return strconv.FormatInt(*m.Delta, 10)
}
