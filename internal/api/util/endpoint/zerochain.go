package endpoint

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

type CallNode func(node string) (*resty.Response, error)
type ConsensusMetFunction func(response *resty.Response, resolvedObject interface{}) bool

func ConsensusByHttpStatus(expectedStatus string) ConsensusMetFunction {
	return func(response *resty.Response, resolvedObject interface{}) bool {
		return response.Status() == expectedStatus
	}
}

func (z *Zerochain) Init(networkEntrypoint string) {
	z.restClient = *resty.New() //nolint
	resp, err := z.restClient.R().Get(networkEntrypoint)
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

func (z *Zerochain) GetFromMiners(t *testing.T, endpoint string, consensusMet ConsensusMetFunction, targetObject interface{}) (*resty.Response, error) { //nolint
	getFromMiner := func(miner string) (*resty.Response, error) {
		return z.GetFromMiner(t, miner, endpoint, targetObject)
	}
	return z.executeWithConsensus(t, z.Miners, getFromMiner, targetObject, consensusMet)
}

func (z *Zerochain) GetFromMiner(t *testing.T, miner, endpoint string, targetObject interface{}) (*resty.Response, error) { //nolint
	resp, err := z.restClient.R().Get(miner + endpoint)
	if resp != nil && resp.IsError() {
		t.Logf("GET on miner [" + miner + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("GET on miner [" + miner + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		t.Logf("GET on miner [" + miner + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
		unmarshalError := json.Unmarshal(resp.Body(), targetObject)

		if unmarshalError != nil {
			return resp, unmarshalError
		}

		return resp, nil
	}
}

func (z *Zerochain) PostToMiners(t *testing.T, endpoint string, consensusMet ConsensusMetFunction, body interface{}, targetObject interface{}) (*resty.Response, error) { //nolint
	postToMiner := func(miner string) (*resty.Response, error) {
		return z.PostToMiner(t, miner, endpoint, body, targetObject)
	}
	return z.executeWithConsensus(t, z.Miners, postToMiner, targetObject, consensusMet)
}

func (z *Zerochain) PostToMiner(t *testing.T, miner, endpoint string, body interface{}, targetObject interface{}) (*resty.Response, error) { //nolint
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

func (z *Zerochain) PostToShardersWithFormData(t *testing.T, endpoint string, consensusMet ConsensusMetFunction, formData map[string]string, body interface{}, targetObject interface{}) (*resty.Response, error) { //nolint
	postToSharder := func(sharder string) (*resty.Response, error) {
		return z.PostToSharder(t, sharder, endpoint, formData, body, targetObject)
	}
	return z.executeWithConsensus(t, z.Sharders, postToSharder, targetObject, consensusMet)
}

func (z *Zerochain) PostToSharder(t *testing.T, sharder, endpoint string, formData map[string]string, body interface{}, targetObject interface{}) (*resty.Response, error) { //nolint
	resp, err := z.restClient.R().SetFormData(formData).SetBody(body).Post(sharder + endpoint)

	if resp != nil && resp.IsError() {
		t.Logf("POST on sharder [" + sharder + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("POST on sharder [" + sharder + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		t.Logf("POST on sharder [" + sharder + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
		unmarshalError := json.Unmarshal(resp.Body(), targetObject)

		if unmarshalError != nil {
			return resp, unmarshalError
		}

		return resp, nil
	}
}

func (z *Zerochain) PostToBlobber(t *testing.T, blobber, endpoint string, headers, formData map[string]string, body []byte, targetObject interface{}) (*resty.Response, error) { //nolint
	resp, err := z.restClient.R().
		SetHeaders(headers).
		SetFormData(formData).
		SetBody(body).
		Post(blobber + endpoint)

	if resp != nil && resp.IsError() {
		t.Logf("POST on blobber [" + blobber + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("POST on blobber [" + blobber + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		t.Logf("POST on blobber [" + blobber + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
		unmarshalError := json.Unmarshal(resp.Body(), targetObject)

		if unmarshalError != nil {
			return resp, unmarshalError
		}

		return resp, nil
	}
}

func (z *Zerochain) GetFromBlobber(t *testing.T, blobber, endpoint string, headers, params map[string]string, targetObject interface{}) (*resty.Response, error) { //nolint
	resp, err := z.restClient.R().
		SetHeaders(headers).
		SetQueryParams(params).
		Get(blobber + endpoint)

	if resp != nil && resp.IsError() {
		t.Logf("GET on blobber [" + blobber + "] endpoint [" + endpoint + "] was unsuccessful, resulting in HTTP [" + resp.Status() + "] and body [" + resp.String() + "]")
		return resp, nil
	} else if err != nil {
		t.Logf("GET on blobber [" + blobber + "] endpoint [" + endpoint + "] processed with error [" + err.Error() + "]")
		return resp, err
	} else {
		t.Logf("GET on blobber [" + blobber + "] endpoint [" + endpoint + "] processed without error, resulting in HTTP [" + resp.Status() + "] with body [" + resp.String() + "]")
		unmarshalError := json.Unmarshal(resp.Body(), targetObject)

		if unmarshalError != nil {
			return resp, unmarshalError
		}

		return resp, nil
	}
}

func (z *Zerochain) GetFromSharders(t *testing.T, endpoint string, consensusMet ConsensusMetFunction, targetObject interface{}) (*resty.Response, error) { //nolint
	getFromSharder := func(sharder string) (*resty.Response, error) {
		return z.GetFromSharder(t, sharder, endpoint, targetObject)
	}
	return z.executeWithConsensus(t, z.Sharders, getFromSharder, targetObject, consensusMet)
}

func (z *Zerochain) GetFromSharder(t *testing.T, sharder string, endpoint string, targetObject interface{}) (*resty.Response, error) { //nolint
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

func (z *Zerochain) executeWithConsensus(t *testing.T, nodes []string, callNode CallNode, targetObject interface{}, consensusMet ConsensusMetFunction) (*resty.Response, error) {
	errors := make([]error, 0)
	responsesAsExpected := make([]*resty.Response, 0)
	responsesNotAsExpected := make([]*resty.Response, 0)

	for _, node := range nodes {
		httpResponse, httpError := callNode(node)

		if httpError != nil {
			errors = append(errors, httpError)
		}

		if httpResponse != nil {
			if consensusMet == nil || consensusMet(httpResponse, targetObject) {
				responsesAsExpected = append(responsesAsExpected, httpResponse)
			} else {
				responsesNotAsExpected = append(responsesNotAsExpected, httpResponse)
			}
		}
	}

	errorSize := float64(len(errors))
	responsesAsExpectedSize := float64(len(responsesAsExpected))
	responsesNotAsExpectedSize := float64(len(responsesNotAsExpected))

	t.Logf("Consensus for operation was [%.2f%%] HTTP response as expeted, [%.2f%%] HTTP response NOT as expexted, [%.2f%%] error", (float64(100)/(responsesAsExpectedSize+responsesNotAsExpectedSize+errorSize))*responsesAsExpectedSize, (float64(100)/(responsesAsExpectedSize+responsesNotAsExpectedSize+errorSize))*responsesNotAsExpectedSize, (float64(100)/(responsesAsExpectedSize+responsesNotAsExpectedSize+errorSize))*errorSize)

	if errorSize > responsesAsExpectedSize+responsesNotAsExpectedSize {
		return nil, mostDominantError(errors)
	}

	if responsesNotAsExpectedSize > responsesAsExpectedSize {
		return responsesNotAsExpected[0], nil
	}

	return responsesAsExpected[0], nil
}

func mostDominantError(errors []error) error {
	var mostFrequent error
	topFrequencyCount := 0

	for _, currentError := range errors {
		currentFrequencyCount := 0
		for _, compareToError := range errors {
			if currentError == compareToError {
				currentFrequencyCount++
			}
		}

		if currentFrequencyCount > topFrequencyCount {
			topFrequencyCount = currentFrequencyCount
			mostFrequent = currentError
		}
	}

	return mostFrequent
}
