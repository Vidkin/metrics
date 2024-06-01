package main

import (
	"fmt"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	var memStorage = storage.New()
	pattern := fmt.Sprintf("/update/{%s}/{%s}/{%s}", internal.ParamMetricType, internal.ParamMetricName, internal.ParamMetricValue)

	mux.HandleFunc(pattern, handlers.MetricsHandler(memStorage))
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		panic(err)
	}
}
