package sdk_tests

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
)

var (
	apiClient *client.APIClient
	sdkClient *client.SDKClient

	parsedConfig *config.Config
)

func TestMain(m *testing.M) {
	// Setup code here (runs ONCE before all tests)
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = "./config/sdk_tests_config.yaml"
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}
	parsedConfig = config.Parse(configPath)
	apiClient = client.NewAPIClient(parsedConfig.BlockWorker)
	sdkClient = client.NewSDKClient(parsedConfig.BlockWorker)

	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		test.SmokeTestMode, _ = strconv.ParseBool(os.Getenv("SMOKE_TEST_MODE"))
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	exitCode := m.Run() // This runs all your Test* functions

	// Cleanup code here (runs ONCE after all tests)
	os.Exit(exitCode)
}
