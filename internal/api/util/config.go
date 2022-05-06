package util

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NetworkEntrypoint string `yaml:"network_entrypoint"`
}

func (config *Config) Init(path string) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Failed to read config file!")
	}
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		panic("failed to deserialise config file!")
	}
}
