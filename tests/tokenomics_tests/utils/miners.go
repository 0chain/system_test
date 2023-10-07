package utils

import (
	"encoding/json"
	"fmt"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

const (
	MinerSmartContractAddress   = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"
	FaucetSmartContractAddress  = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d3"
	StorageSmartContractAddress = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	ZCNSmartContractAddess      = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712e0"
)

func apiGetLatestFinalized(sharderBaseURL string) (*http.Response, error) {
	return http.Get(sharderBaseURL + "/v1/block/get/latest_finalized")
}

func getNodeBaseURL(host string, port int) string {
	return fmt.Sprintf(`http://%s:%d`, host, port)
}

func GetLatestFinalizedBlock(t *test.SystemTest) *climodel.LatestFinalizedBlock {
	output, err := CreateWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	sharders := getShardersList(t)
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	sharderBaseUrl := getNodeBaseURL(sharder.Host, sharder.Port)

	res, err := apiGetLatestFinalized(sharderBaseUrl)
	require.Nil(t, err, "Error retrieving latest block")
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get latest block: %d", res.StatusCode)
	require.NotNil(t, res.Body, "Latest block API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var block climodel.LatestFinalizedBlock
	err = json.Unmarshal(resBody, &block)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return &block
}

func GetSortedMinerIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getMinerList", sharderBaseURL)
}

func getSortedNodeIds(t *test.SystemTest, endpoint, sharderBaseURL string) []string {
	nodeList := getNodeSlice(t, endpoint, sharderBaseURL)
	var nodeIds []string
	for i := range nodeList {
		nodeIds = append(nodeIds, nodeList[i].ID)
	}
	sort.Slice(nodeIds, func(i, j int) bool {
		return nodeIds[i] < nodeIds[j]
	})
	return nodeIds
}

func getNodeSlice(t *test.SystemTest, endpoint, sharderBaseURL string) []climodel.Node {
	t.Logf("getting miner or sharder nodes...")
	url := sharderBaseURL + "/v1/screst/" + MinerSmartContractAddress + "/" + endpoint
	nodeList := cliutil.ApiGetRetries[climodel.NodeList](t, url, nil, 3)
	return nodeList.Nodes
}

func MinerOrSharderLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderLockForWallet(t, cliConfigFilename, params, EscapedTestName(t), retry)
}

func minerOrSharderLockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("locking tokens against miner/sharder...")
	if retry {
		return cliutil.RunCommand(t, fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutil.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
