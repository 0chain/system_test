package cli_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0chain/gosdk/core/conf"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
)

var configPath string

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		configPath = "./zbox_config.yaml"
		cli_utils.Logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}

	if !strings.EqualFold(strings.TrimSpace(os.Getenv("SKIP_CONFIG_CLEANUP")), "true") {
		if files, err := filepath.Glob("./config/*.json"); err == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}

		if files, err := filepath.Glob("./config/*.txt"); err == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}
	}

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
