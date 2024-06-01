package main

import (
	"github.com/Vidkin/metrics/internal/domain/storage"
	"runtime"
)

func main() {
	var memoryStorage = storage.New()
	memStats := &runtime.MemStats{}

	Poll(memoryStorage, memStats)
}
