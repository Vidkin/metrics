package metric

import "strconv"

type Metric struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func (m *Metric) ValueAsString() string {
	if m.MType == "gauge" {
		return strconv.FormatFloat(*m.Value, 'g', -1, 64)
	}
	return strconv.FormatInt(*m.Delta, 10)
}
