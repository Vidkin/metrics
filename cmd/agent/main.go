package main

import (
	"github.com/Vidkin/metrics/internal/domain/storage"
	"github.com/go-resty/resty/v2"
	"runtime"
)

func main() {
	var memoryStorage = storage.New()
	memStats := &runtime.MemStats{}
	client := resty.New()

	Poll(client, memoryStorage, memStats)
}
