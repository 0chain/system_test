package util

import (
	"encoding/json"
	"testing"

	resty "github.com/go-resty/resty/v2"
)

type Zerochain struct {
	Miners     []string     `json:"miners"`
	Sharders   []string     `json:"sharders"`
	restClient resty.Client //nolint
}

func (z *Zerochain) Init(config Config) {
	z.restClient = *resty.New() //nolint
	resp, err := z.restClient.R().Get(config.NetworkEntrypoint)
	if err != nil {
		panic("0dns call failed!: encountered error [" + err.Error() + "]")
	}

	err = json.Unmarshal(resp.Body(), z)
	if err != nil {
		panic("0dns call failed!: encountered error [" + err.Error() + "] when trying to serialize body [" + resp.String() + "]")
	}

	healthyMiners, healthySharders := z.performHealthcheck()
	z.Miners = healthyMiners
	z.Sharders = healthySharders
}

func (z *Zerochain) getRandomMiner() string {
	return z.Miners[0]
}

func (z *Zerochain) getRandomSharder() string {
	return z.Sharders[0]
}

func (z *Zerochain) getFromMiners(t *testing.T, endpoint string) (*resty.Response, error) { //nolint
	miner := z.getRandomMiner()
	resp, err := z.restClient.R().Get(miner + endpoint)
	t.Logf("GET on miner [" + miner + "] endpoint  [" + endpoint + "] resulted in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")

	return resp, err
}

func (z *Zerochain) PostToMiners(t *testing.T, endpoint string, body interface{}, targetObject interface{}) (*resty.Response, error) { //nolint
	miner := z.getRandomMiner()
	resp, err := z.restClient.R().SetBody(body).Post(miner + endpoint)

	if resp != nil && resp.IsError() {
		t.Logf("POST on miner [" + miner + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("POST on miner [" + miner + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		t.Logf("POST on miner [" + miner + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
		unmarshalError := json.Unmarshal(resp.Body(), targetObject)

		if unmarshalError != nil {
			return resp, unmarshalError
		}

		return resp, nil
	}
}

func (z *Zerochain) GetFromSharders(t *testing.T, endpoint string, targetObject interface{}) (*resty.Response, error) { //nolint
	sharder := z.getRandomSharder()
	resp, err := z.restClient.R().Get(sharder + endpoint)

	if resp != nil && resp.IsError() {
		t.Logf("GET on sharder [" + sharder + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("GET on sharder [" + sharder + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		if targetObject != nil {
			t.Logf("GET on sharder [" + sharder + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
			unmarshalError := json.Unmarshal(resp.Body(), targetObject)
			if unmarshalError != nil {
				return resp, unmarshalError
			}
			return resp, nil
		}
		return resp, nil
	}
}

func (z *Zerochain) performHealthcheck() ([]string, []string) {
	healthyMiners := z.getHealthyNodes(z.Miners)
	healthySharders := z.getHealthyNodes(z.Sharders)

	if len(healthyMiners) == 0 {
		panic("No healthy miners found!")
	}
	if len(healthySharders) == 0 {
		panic("No healthy sharders found!")
	}

	return healthyMiners, healthySharders
}

func (z *Zerochain) getHealthyNodes(nodes []string) []string {
	var healthyNodes []string
	for _, node := range nodes {
		healthResponse, err := z.restClient.R().Get(node + "/v1/chain/get/stats")

		if err == nil && healthResponse.IsSuccess() {
			println(node + " is UP!")
			healthyNodes = append(healthyNodes, node)
		} else {
			println(node + " is DOWN!")
		}
	}

	return healthyNodes
}
