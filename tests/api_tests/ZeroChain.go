package api_tests

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"testing"
)

var restClient resty.Client

type Zerochain struct {
	Miners   []string `json:"miners"`
	Sharders []string `json:"sharders"`
}

func (zerochain *Zerochain) init(config Config) {
	restClient = *resty.New()
	resp, err := restClient.R().Get(config.NetworkEntrypoint)

	err = json.Unmarshal(resp.Body(), zerochain)
	if err != nil {
		panic("0dns call failed!: encountered error [" + err.Error() + "] when trying to serialise body [" + resp.String() + "]")
	}

	healthyMiners, healthySharders := zerochain.performHealthcheck()
	zerochain.Miners = healthyMiners
	zerochain.Sharders = healthySharders
}

func (zerochain *Zerochain) getRandomMiner() string {
	return zerochain.Miners[0]
}

func (zerochain *Zerochain) getRandomSharder() string {
	return zerochain.Sharders[0]
}

func (zerochain *Zerochain) getFromMiners(t *testing.T, endpoint string) (*resty.Response, error) {
	miner := zerochain.getRandomMiner()
	resp, err := restClient.R().Get(miner + endpoint)
	t.Logf("GET on miner [" + miner + "] endpoint  [" + endpoint + "] resulted in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")

	return resp, err
}

func (zerochain *Zerochain) getFromSharders(t *testing.T, endpoint string) (*resty.Response, error) {
	sharder := zerochain.getRandomSharder()
	resp, err := restClient.R().Get(sharder + endpoint)
	t.Logf("GET on sharder [" + sharder + "] endpoint  [" + endpoint + "] resulted in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")

	return resp, err
}

func (zerochain *Zerochain) performHealthcheck() ([]string, []string) {
	healthyMiners := getHealthyNodes(zerochain.Miners)
	healthySharders := getHealthyNodes(zerochain.Sharders)

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
