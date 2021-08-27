package tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/config"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
	"os"
	"testing"
)

var configPath string

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")

	if configPath == ""{
		fmt.Print("Config_Path is not set")
		panic("CONFIG_PATH must be passed")
	}

	log.SetLevel(log.ErrorLevel)
	exitRun := m.Run()
	os.Exit(exitRun)
}

func GetConfig(t *testing.T) *config.RequiredConfig {
	t.Helper()
	if configPath == "" {
		t.Fatal("configPath is empty, TestMain not called")
	}

	configurer, err := config.NewConfigurer(configPath)
	if err != nil {
		t.Fatalf("failed to fetch configuration from the ConfigPath: %v", err)
	}

	requiredConfig := configurer.RequiredConfig

	validate := validator.New()
	if err := validate.Struct(requiredConfig); err != nil {
		t.Fatalf("failed to get configuration from the configFile: %v", err)
	}

	return requiredConfig
}
