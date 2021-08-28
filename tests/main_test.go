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
var requiredConfig config.RequiredConfig

func TestMain(m *testing.M) {
	configPath = os.Getenv("CONFIG_PATH")

	if configPath == "" {
		fmt.Print("Config_Path is not set")
		panic("CONFIG_PATH must be passed")
	}

	requiredConfig = *GetConfig()

	level, _ := log.ParseLevel(*requiredConfig.LogLevel)
	log.SetLevel(level)

	exitRun := m.Run()

	os.Exit(exitRun)
}

func GetConfig() *config.RequiredConfig {
	if configPath == "" {
		panic("configPath is empty, TestMain not called")
	}

	configurer, err := config.NewConfigurer(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to fetch configuration from the ConfigPath: %v", err))
	}

	requiredConfig := configurer.RequiredConfig

	validate := validator.New()
	if err := validate.Struct(requiredConfig); err != nil {
		panic(fmt.Sprintf("failed to get configuration from the configFile: %v", err))
	}

	return requiredConfig
}
