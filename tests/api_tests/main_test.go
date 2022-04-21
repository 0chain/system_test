package api_tests

import (
	"encoding/json"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/go-resty/resty/v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

var (
	config     Config
	network    Network
	restClient *resty.Client
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
		cliutils.Logger.Infof("CONFIG_PATH environment variable is not set so has defaulted to [%v]", configPath)
	}
	config = getConf(configPath)

	restClient = resty.New()

	resp, err := restClient.R().Get(config.NetworkEntrypoint)

	var network Network
	err = json.Unmarshal(resp.Body(), &network)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}

	performHealthcheck(network)
}

func performHealthcheck(network Network) {
	for _, miner := range network.Miners {
		healthResponse, err := restClient.R().Get(miner + "/v1/chain/get/stats")

		if err == nil && healthResponse.IsSuccess() {
			println(miner + " is success!")
		} else {
			println(miner + " is NOT success!")
		}
	}

	for _, sharder := range network.Sharders {
		healthResponse, err := restClient.R().Get(sharder + "/v1/chain/get/stats")

		if err == nil && healthResponse.IsSuccess() {
			println(sharder + " is success!")
		} else {
			println(sharder + " is NOT success!")
		}
	}
}

func getConf(path string) Config {
	var config Config
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("ERR0R: %v", err)
	}

	return config
}
