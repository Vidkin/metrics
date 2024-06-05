package main

import (
	"flag"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"net/http"
)

func main() {
	addr := new(ServerAddress)
	addr.Host = internal.DefaultServerAddress
	addr.Port = internal.DefaultServerPort

	flag.Var(addr, "a", "Net address host:port")
	flag.Parse()

	var memStorage = storage.New()
	err := http.ListenAndServe(addr.String(), handlers.MetricsRouter(memStorage))

	if err != nil {
		panic(err)
	}
}
