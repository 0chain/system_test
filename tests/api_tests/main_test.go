package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"log"
	"os"
	"testing"
)

var apiClient *client.APIClient

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig := config.Parse(configPath)

	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)

	os.Exit(m.Run())
}
