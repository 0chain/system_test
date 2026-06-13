package sdk_tests

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"runtime"

	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/core/conf"

	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
)

func TestMain(m *testing.M) {
	configPath, ok := os.LookupEnv(config.ConfigPathEnv)
	if !ok {
		configPath = "./config/sdk_tests_config.yaml"
		log.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	// Parse configuration
	parsedConfig = parseConfig(configPath)

	defaultTestTimeout, err := time.ParseDuration(parsedConfig.DefaultTestTimeout)
	if err != nil {
		log.Printf("Default test case timeout could not be parsed so has defaulted to [%v]", test.DefaultTestTimeout)
	} else {
		test.DefaultTestTimeout = defaultTestTimeout
		test.SmokeTestMode, _ = strconv.ParseBool(os.Getenv("SMOKE_TEST_MODE"))
		log.Printf("Default test case timeout is [%v]", test.DefaultTestTimeout)
	}


	// t := test.NewSystemTest(new(testing.T))

	// Initialize the SDK client
	// Initialize the SDK client
	err = client.Init(context.Background(), conf.Config{
		BlockWorker:     parsedConfig.BlockWorker,
		SignatureScheme: parsedConfig.SignatureScheme,
		ChainID:         parsedConfig.ChainID,
		MaxTxnQuery:     parsedConfig.MaxTxnQuery,
		QuerySleepTime:  parsedConfig.QuerySleepTime,
		MinSubmit:       parsedConfig.MinSubmit,
		MinConfirmation: parsedConfig.MinConfirmation,
	})
	// require.NoError(t, err, "Failed to initialize SDK client")
	if err != nil {
		log.Printf("Failed to initialize SDK client: %v", err)
		runtime.Breakpoint() // Force debugger to pause
		os.Exit(1)
	}
	client.SetSdkInitialized(true)
	client.SetSignatureScheme(parsedConfig.SignatureScheme)

	// Load wallet pool if available
	walletMutex.Lock()
	walletFilePath := filepath.Join(".", "config", "wallets.json")
	if _, err := os.Stat(walletFilePath); err == nil {
		fileContent, err := os.ReadFile(walletFilePath)
		if err != nil {
			log.Printf("Warning: Could not read wallets file: %v", err)
		} else {
			err = json.Unmarshal(fileContent, &initialisedWallets)
			if err != nil {
				log.Printf("Warning: Could not parse wallets file: %v", err)
			} else {
				log.Printf("Loaded %d wallets from pool", len(initialisedWallets))
			}
		}
	} else {
		log.Printf("No wallet pool found at %s, tests will create wallets as needed", walletFilePath)
	}
	walletMutex.Unlock()

	os.Exit(m.Run())
}

func parseConfig(configPath string) *SDKTestConfig {
	cfg := &SDKTestConfig{
		SignatureScheme: "bls0chain",
		ChainID:         "0afc093ffb509f059c55478bc1a60351cef7b4e9c008a53a6cc8241ca8617dfe",
		MaxTxnQuery:     5,
		QuerySleepTime:  5,
		MinSubmit:       10,
		MinConfirmation: 10,
		DefaultTestTimeout: "45s",
	}

	parsed := config.Parse(configPath)
	if parsed != nil {
		// Map common fields
		cfg.BlockWorker = parsed.BlockWorker
		cfg.DefaultTestTimeout = parsed.DefaultTestCaseTimeout
	}

	return cfg
}
