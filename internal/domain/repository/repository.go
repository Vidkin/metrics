package repository

type Repository interface {
	UpdateGauge(key string, value float64)
	UpdateCounter(key string, value int64)

	GetGauges() map[string]float64
	GetCounters() map[string]int64
}
