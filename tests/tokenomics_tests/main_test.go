package tokenomics_tests

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/tokenomics/util/config"
)

var (
	parsedConfig *config.Config
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = config.DefaultConfigPath
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	parsedConfig = config.Parse(configPath)

	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestCaseTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}

	os.Exit(m.Run())
}
