package api_tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestGetLatestFinalizedMagicBlock(t *testing.T) {
	t.Parallel()

	t.Run("Lfmb node hash not modified, should return http 304 and empty body", func(t *testing.T) {
		t.Parallel()

		hash, err := getCurrentHash(t)
		require.Nil(t, err)

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, hash, http.StatusNotModified)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusNotModified)
		require.Nil(t, err, string(resp.Body()))

		require.Empty(t, string(resp.Body()))
	})

	t.Run("No param provided, should return http 200 and current return whole lfmb message as before", func(t *testing.T) {
		t.Parallel()

		hash, err := getCurrentHash(t)
		require.Nil(t, err)

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, "", http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err, string(resp.Body()))

		require.NotEmpty(t, string(resp.Body()))
		var res map[string]interface{}
		json.Unmarshal(resp.Body(), &res)
		require.Equal(t, hash, res["hash"])
	})

	t.Run("Different node-lfmb-hash provided, return http 200 and return the lfmb the sharder has", func(t *testing.T) {
		t.Parallel()

		hash, err := getCurrentHash(t)
		require.Nil(t, err)

		hash = "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae5"

		resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, hash, http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err, string(resp.Body()))

		require.NotEmpty(t, string(resp.Body()))

		var res map[string]interface{}
		json.Unmarshal(resp.Body(), &res)

		require.NotEqual(t, hash, res["hash"])
	})
}

func getCurrentHash(t *testing.T) (string, error) {

	resp, err := http.Get("https://test.0chain.net/sharder01/v1/block/get/latest_finalized_magic_block")

	if err != nil {
		return "", err
	}

	var res map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&res)

	if err != nil {
		return "", err
	}

	resultHash := res["hash"]

	strHash := fmt.Sprintf("%s", resultHash)

	return strHash, nil
}
