package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/setup"
	"log"
	"os"
	"testing"
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

	setup.InitTestEnvironment(configPath)

	os.Exit(m.Run())
}
