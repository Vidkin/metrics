// Package logger provides a simple logging utility using the Zap logging library.
package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(logLevel string) error {
	lvl, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	logger, err := cfg.Build()
	if err != nil {
		return err
	}
	defer logger.Sync()

	Log = logger
	return nil
}
