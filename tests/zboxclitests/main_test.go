package zboxclitests

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
		fmt.Print("Config_Path is not set")
		panic("CONFIG_PATH must be passed")
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
