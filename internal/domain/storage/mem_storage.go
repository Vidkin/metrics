package storage

type MemStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func New() *MemStorage {
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

func (m *MemStorage) GetGauges() map[string]float64 {
	return m.Gauge
}

func (m *MemStorage) GetCounters() map[string]int64 {
	return m.Counter
}
