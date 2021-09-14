package cli_tests

import (
	"fmt"
	"github.com/0chain/gosdk/core/conf"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

var configPath string

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./zbox_config.yaml"
		fmt.Printf("CONFIG_PATH environment variable is not set so has defaulted to [%s]\n", configPath)
	}

	log.SetLevel(log.ErrorLevel)
	exitRun := m.Run()
	os.Exit(exitRun)
}

func GetConfig(t *testing.T) conf.Config {
	t.Helper()
	if configPath == "" {
		t.Fatal("configPath is empty, TestMain not called")
	}

	config, err := conf.LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("failed to fetch configuration from the ConfigPath: %v", err)
	}

	return config
}
