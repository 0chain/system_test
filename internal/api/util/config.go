package util

import (
	"io/ioutil"

	"gopkg.in/yaml.v3" //nolint
)

type Config struct {
	NetworkEntrypoint string `yaml:"network_entrypoint"`
}

func (config *Config) Init(path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Failed to read config file!")
	}
	err = yaml.Unmarshal(yamlFile, config) //nolint
	if err != nil {
		panic("failed to deserialise config file!")
	}
}
