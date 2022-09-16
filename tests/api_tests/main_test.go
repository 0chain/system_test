package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/endpoint"

	"log"
	"os"
	"testing"
)

var zeroChain endpoint.Zerochain

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig := config.Parse(configPath)

	zeroChain.Init(parsedConfig.NetworkEntrypoint)

	os.Exit(m.Run())
}
