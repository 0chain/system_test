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
	BlockWorker            string `yaml:"block_worker"`
	ZboxUrl                string `yaml:"0box_url"`
	ZboxPhoneNumber        string `yaml:"0box_phone_number"`
	DefaultTestCaseTimeout string `yaml:"default_test_case_timeout"`
	ZS3ServerUrl           string `yaml:"zs3_server_url"`
	S3SecretKey            string `yaml:"s3_secret_key"`
	S3AccessKey            string `yaml:"s3_access_key"`
	S3BucketName           string `yaml:"s3_bucket_name"`
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
