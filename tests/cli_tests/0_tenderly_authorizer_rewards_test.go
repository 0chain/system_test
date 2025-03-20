package cli_tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/cli/model"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func Test0TenderlyAuthorizerRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if !tenderlyInitialized {
		t.Skip("Tenderly has not been initialized properly!")
	}

	t.RunSequentiallyWithTimeout("Verify Authorizer Rewards", time.Minute*10, func(t *test.SystemTest) {
		time.Sleep(time.Minute)

		createWallet(t)

		feeRewardAuthorizerQuery := fmt.Sprintf("reward_type = %d", model.FeeRewardAuthorizer)
		feeRewardAuthorizer, err := getQueryRewards(t, feeRewardAuthorizerQuery)
		require.Nil(t, err)

		output, err := burnEth(t, "1000000000000", true)
		require.Nil(t, err)
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Verification:")

		output, err = mintZcnTokens(t, true)
		require.Nil(t, err, "error: %s", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Done.")

		time.Sleep(20 * time.Second)

		feeRewardAuthorizerAfterMint, err := getQueryRewards(t, feeRewardAuthorizerQuery)
		require.Nil(t, err)

		require.Equal(t, feeRewardAuthorizerAfterMint.TotalReward, feeRewardAuthorizer.TotalReward+33, "Fee reward authorizer should be increased by 33.33 ZCN")
	})
}

func getQueryRewards(t *test.SystemTest, query string) (QueryRewardsResponse, error) {
	var result QueryRewardsResponse

	StorageScAddress := "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7"
	sharderBaseUrl := utils.GetSharderUrl(t)
	requestURL := fmt.Sprintf("%s/v1/screst/%s/query-rewards?query=%s",
		sharderBaseUrl, StorageScAddress, url.QueryEscape(query))

	res, _ := http.Get(requestURL) //nolint:gosec

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)

	body, _ := io.ReadAll(res.Body)

	err := json.Unmarshal(body, &result)
	if err != nil {
		return QueryRewardsResponse{}, err
	}

	return result, nil
}

type QueryRewardsResponse struct {
	TotalProviderReward float64 `json:"total_provider_reward"`
	TotalDelegateReward float64 `json:"total_delegate_reward"`
	TotalReward         float64 `json:"total_reward"`
}
