package internal

type MemStorage struct {
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	var m MemStorage
	m.gauge = make(map[string]float64)
	m.counter = make(map[string]int64)
	return &m
}

func (m *MemStorage) UpdateGauge(key string, value float64) {
	m.gauge[key] = value
}

func (m *MemStorage) UpdateCounter(key string, value int64) {
	m.counter[key] += value
}
