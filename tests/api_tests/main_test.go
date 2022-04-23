package api_tests

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var (
	config     Config
	network    Network
	restClient = resty.New()
	logger     = getLogger()
)

type Config struct {
	NetworkEntrypoint string `yaml:"network_entrypoint"`
}

type Network struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

func TestMain(m *testing.M) {
	var configPath = os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/api_tests_config.yaml"
		logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}
	config = getConf(configPath)

	resp, err := restClient.R().Get(config.NetworkEntrypoint)

	err = json.Unmarshal(resp.Body(), &network)
	if err != nil {
		panic("0dns call failed!")
	}

	healthyMiners, healthySharders := performHealthcheck(network)
	network.Miners = healthyMiners
	network.Sharders = healthySharders

	exitRun := m.Run()
	os.Exit(exitRun)
}

func performHealthcheck(network Network) ([]string, []string) {
	healthyMiners := getHealthyNodes(network.Miners)
	healthySharders := getHealthyNodes(network.Sharders)

	if len(healthyMiners) == 0 {
		panic("No healthy miners found!")
	}
	if len(healthySharders) == 0 {
		panic("No healthy sharders found!")
	}

	return healthyMiners, healthySharders
}

func getHealthyNodes(nodes []string) []string {
	var healthyNodes []string
	for _, node := range nodes {
		healthResponse, err := restClient.R().Get(node + "/v1/chain/get/stats")

		if err == nil && healthResponse.IsSuccess() {
			println(node + " is UP!")
			healthyNodes = append(healthyNodes, node)
		} else {
			println(node + " is DOWN!")
		}
	}

	return healthyNodes
}

func getConf(path string) Config {
	var config Config
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Failed to read config file!")
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic("failed to deserialise config file!")
	}

	return config
}

func getLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout

	logger.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEBUG")), "true") {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}
