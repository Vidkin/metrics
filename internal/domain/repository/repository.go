package repository

type Repository interface {
	UpdateGauge(key string, value float64)
	UpdateCounter(key string, value int64)

	GetGauges() map[string]float64
	GetCounters() map[string]int64

	GetGauge(metricName string) (float64, bool)
	GetCounter(metricName string) (int64, bool)
}
