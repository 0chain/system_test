package api_tests

import (
	"os"
	"testing"
)

var (
	config         Config
	zeroChain      Zerochain
	fallbackLogger FallbackLogger
)

func TestMain(m *testing.M) {
	fallbackLogger.init()
	var configPath = os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/api_tests_config.yaml"
		fallbackLogger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}
	config.init(configPath)
	zeroChain.init(config)

	exitRun := m.Run()
	os.Exit(exitRun)
}
