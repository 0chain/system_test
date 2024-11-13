package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3" //nolint
)

// ConfigPathEnv contains name of env variable
const ConfigPathEnv = "CONFIG_PATH"

// DefaultConfigPath contains default value of ConfigPathEnv
const DefaultConfigPath = "./config/tokenomics_tests_config.yaml"

type Config struct {
	BlockWorker            string `yaml:"block_worker"`
	ZboxUrl                string `yaml:"0box_url"`
	ZboxPhoneNumber        string `yaml:"0box_phone_number"`
	DefaultTestCaseTimeout string `yaml:"default_test_case_timeout"`
	ZS3ServerUrl           string `yaml:"zs3_server_url"`
	ChimneyTestNetwork     string `yaml:"chimney_test_network"`
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
