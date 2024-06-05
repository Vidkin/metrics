package main

import (
	"flag"
	"github.com/Vidkin/metrics/internal"
	"github.com/Vidkin/metrics/internal/domain/handlers"
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/caarlos0/env/v6"
	"net/http"
)

func main() {
	addr := new(ServerAddress)
	addr.Host = internal.DefaultServerAddress
	addr.Port = internal.DefaultServerPort

	flag.Var(addr, "a", "Net address host:port")
	flag.Parse()

	env.Parse(addr)

	var memStorage = storage.New()

	var err error
	if addr.Address != "" {
		err = http.ListenAndServe(addr.Address, handlers.MetricsRouter(memStorage))
	} else {
		err = http.ListenAndServe(addr.String(), handlers.MetricsRouter(memStorage))
	}

	if err != nil {
		panic(err)
	}
}
