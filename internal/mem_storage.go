package internal

type MemStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func NewMemStorage() *MemStorage {
	var m MemStorage
	m.Gauge = make(map[string]float64)
	m.Counter = make(map[string]int64)
	return &m
}

func (m *MemStorage) UpdateGauge(key string, value float64) {
	m.Gauge[key] = value
}

func (m *MemStorage) UpdateCounter(key string, value int64) {
	m.Counter[key] += value
}
