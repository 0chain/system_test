package utils

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"reflect"
	"strings"
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
