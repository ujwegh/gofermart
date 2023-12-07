package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func InitLogger(level string) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		panic(err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	Log = zl
}
