package api_tests

import (
	"log"
	"os"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
)

var (
	apiClient *client.APIClient
	sdkClient *client.SDKClient
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig := config.Parse(configPath)

	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)

	os.Exit(m.Run())
}
