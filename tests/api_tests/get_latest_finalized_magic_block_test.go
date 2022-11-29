package api_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/block"
	"github.com/0chain/gosdk/core/common"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestGetLatestFinalizedMagicBlock(t *testing.T) {
	t.Parallel()

	hash, err := getCurrentHash(t)
	require.Nil(t, err)
	fmt.Println(hash)

	t.Run("Not modified, set node lfmb", func(t *testing.T) {
		t.Parallel()
		lfmb := &block.Block{}
		lfmb.Hash = common.Key(hash)

		aa := model.LatestFinalizedMagicBlock{
			LFMB: lfmb,
		}
		_, resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, &aa, http.StatusNotModified)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusNotModified)
		require.Nil(t, err)

		require.Empty(t, string(resp.Body()))
	})

	t.Run("Not modified, no node lfmb", func(t *testing.T) {
		t.Parallel()

		lfmb := &block.Block{}
		lfmb.Hash = ""

		aa := model.LatestFinalizedMagicBlock{
			LFMB: lfmb,
		}
		_, resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, &aa, http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err)

		require.NotEmpty(t, string(resp.Body()))
		var res map[string]interface{}
		json.Unmarshal(resp.Body(), &res)
		require.Equal(t, hash, res["hash"])
	})

	t.Run("Modified, no node lfmb", func(t *testing.T) {
		t.Parallel()
		lfmb := &block.Block{}
		lfmb.Hash = "ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae5"

		aa := model.LatestFinalizedMagicBlock{
			LFMB: lfmb,
		}
		_, resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, &aa, http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err)

		require.NotEmpty(t, string(resp.Body()))

		var res map[string]interface{}
		json.Unmarshal(resp.Body(), &res)

		require.NotEqual(t, lfmb.Hash, res["hash"])
	})
}

func getCurrentHash(t *testing.T) (string, error) {
	/*
		lfmb := &block.Block{}

		lfmb.Hash = ""
		aa := model.LatestFinalizedMagicBlock{
			LFMB: lfmb,
		}
		_, resp, err := apiClient.V1BlockGetLatestFinalizedMagicBlock(t, &aa, http.StatusOK)
		require.Equal(t, resp.RawResponse.StatusCode, http.StatusOK)
		require.Nil(t, err)
	*/

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
