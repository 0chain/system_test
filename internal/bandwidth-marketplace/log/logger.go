package log

import (
	"github.com/0chain/gosdk/zmagmacore/errors"
	"go.uber.org/zap"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
)

var (
	// Logger represents main logger implementation used in tests.
	Logger *zap.Logger
)

func SetupLogger(cfg config.Log) {
	if !cfg.Enable {
		Logger = zap.NewNop()
		return
	}

	var err error
	Logger, err = zap.NewDevelopment()
	if err != nil {
		errors.ExitErr("Cant create logger.", err, 2)
	}
}
