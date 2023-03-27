package api_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestCoreGetLatestFinalizedMagicBlock(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	t.Run("Lfmb node hash not modified, should return http 304 and empty body", func(t *test.SystemTest) {
		hash, err := getCurrentHash(t)
		require.Nil(t, err)

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, hash, http.StatusNotModified)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusNotModified)
		require.Nil(t, err, string(resp.Body()))

		require.Empty(t, string(resp.Body()))
	})

	t.Run("No param provided, should return http 200 and current return whole lfmb message as before", func(t *test.SystemTest) {
		hash, err := getCurrentHash(t)
		require.Nil(t, err)

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, "", http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err, string(resp.Body()))
		require.NotEmpty(t, string(resp.Body()))

		var res map[string]interface{}
		err = json.Unmarshal(resp.Body(), &res)
		require.Nil(t, err, res)
		require.Equal(t, hash, res["hash"])
	})

	t.Run("Different node-lfmb-hash provided, return http 200 and return the lfmb the sharder has", func(t *test.SystemTest) {
		false_hash := "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae5"

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, false_hash, http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err, string(resp.Body()))

		require.NotEmpty(t, string(resp.Body()))

		var res map[string]interface{}
		err = json.Unmarshal(resp.Body(), &res)
		require.Nil(t, err, res)
		require.NotEqual(t, false_hash, res["hash"])
	})
}

func getCurrentHash(t *test.SystemTest) (string, error) {
	resp, err := http.Get(apiClient.HealthyServiceProviders.Sharders[0] + "/v1/block/get/latest_finalized_magic_block")
	require.Nil(t, err)
	defer resp.Body.Close()

	var res map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&res)
	require.Nil(t, err, res)

	resultHash := res["hash"]

	strHash := fmt.Sprintf("%s", resultHash)

	return strHash, nil
}
