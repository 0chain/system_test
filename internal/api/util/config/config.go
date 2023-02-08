package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3" //nolint
)

// ConfigPathEnv contains name of env variable
const ConfigPathEnv = "CONFIG_PATH"

// DefaultConfigPath contains default value of ConfigPathEnv
const DefaultConfigPath = "./config/api_tests_config.yaml"

type Config struct {
	BlockWorker             string `yaml:"block_worker"`
	EthereumNodeURL         string `yaml:"ethereum_node_url"`
	DefaultTestCaseTimeout  string `yaml:"default_test_case_timeout"`
	DelegatedWalletLocation string `yaml:"delegated_wallet_location"`
	ZboxUrl                string `yaml:"0box_url"`
	ZboxPhoneNumber        string `yaml:"0box_phone_number"`
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

func GetHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}
	return homeDir, nil
}
