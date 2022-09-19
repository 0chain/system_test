package config

import (
	"gopkg.in/yaml.v3" //nolint
	"log"
	"os"
)

// ConfigPathEnv contains name of env variable
const ConfigPathEnv = "CONFIG_PATH"

// DefaultConfigPath contains default value of ConfigPathEnv
const DefaultConfigPath = "./config/api_tests_config.yaml"

type Config struct {
	NetworkEntrypoint string `yaml:"network_entrypoint"`
}

func Parse(configPath string) *Config {
	var result *Config

	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalln("Failed to read config file! due to error: " + err.Error())
	}
	err = yaml.Unmarshal(file, &result) //nolint
	if err != nil {
		log.Fatalln("failed to deserialise config file due to error: " + err.Error())
	}

	return result
}

func MustGetHomeDir() string {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		log.Fatalln(err)
	}
	return homeDir
}
