package internal

const (
	DefaultServerAddress = "localhost"
	DefaultServerPort    = 8080

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"

	ParamMetricType  = "metricType"
	ParamMetricName  = "metricName"
	ParamMetricValue = "metricValue"

	DefaultAgentPollInterval   = 2
	DefaultAgentReportInterval = 10

	GaugeMetricAlloc         = "Alloc"
	GaugeMetricBuckHashSys   = "BuckHashSys"
	GaugeMetricFrees         = "Frees"
	GaugeMetricGCCPUFraction = "GCCPUFraction"
	GaugeMetricGCSys         = "GCSys"
	GaugeMetricHeapAlloc     = "HeapAlloc"
	GaugeMetricHeapIdle      = "HeapIdle"
	GaugeMetricHeapInuse     = "HeapInuse"
	GaugeMetricHeapObjects   = "HeapObjects"
	GaugeMetricHeapReleased  = "HeapReleased"
	GaugeMetricHeapSys       = "HeapSys"
	GaugeMetricLastGC        = "LastGC"
	GaugeMetricLookups       = "Lookups"
	GaugeMetricMCacheInuse   = "MCacheInuse"
	GaugeMetricMCacheSys     = "MCacheSys"
	GaugeMetricMSpanInuse    = "MSpanInuse"
	GaugeMetricMSpanSys      = "MSpanSys"
	GaugeMetricMallocs       = "Mallocs"
	GaugeMetricNextGC        = "NextGC"
	GaugeMetricNumForcedGC   = "NumForcedGC"
	GaugeMetricNumGC         = "NumGC"
	GaugeMetricOtherSys      = "OtherSys"
	GaugeMetricPauseTotalNs  = "PauseTotalNs"
	GaugeMetricStackInuse    = "StackInuse"
	GaugeMetricStackSys      = "StackSys"
	GaugeMetricSys           = "Sys"
	GaugeMetricTotalAlloc    = "TotalAlloc"
	GaugeMetricRandomValue   = "RandomValue"

	CounterMetricPollCount = "PollCount"
)
