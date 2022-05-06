package api_tests

import (
	"os"
	"testing"

	"github.com/0chain/system_test/internal/api/util"
)

var (
	config         util.Config
	zeroChain      util.Zerochain
	fallbackLogger util.FallbackLogger
)

func TestMain(m *testing.M) {
	fallbackLogger.Init()
	var configPath = os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/api_tests_config.yaml"
		fallbackLogger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}
	config.Init(configPath)
	zeroChain.Init(config)

	exitRun := m.Run()
	os.Exit(exitRun)
}
