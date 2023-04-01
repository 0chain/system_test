package cli_tests

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStakePool(testSetup *testing.T) {

	t := test.NewSystemTest(testSetup)

	wallet, err := registerWallet(t, configPath)

	if err != nil {
		return
	}
	fmt.Println(wallet)

	// get the list of blobbers
	blobbersList := getBlobbersList(t)
	require.Len(t, blobbersList, 6, "should have 6 blobbers")

	t.RunSequentiallyWithTimeout("should allow stake pool to be created", 60*time.Minute, func(t *test.SystemTest) {

		expectedWallet1Balance := int64(0)
		expectedWallet2Balance := int64(0)

		// select any random blobber and check total offers
		blobber := blobbersList[0]
		output, _ := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))

		var blInfo BlobberInfo
		err = json.Unmarshal([]byte(output[3]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffers := blInfo.TotalOffers

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates := stakePool.Delegate

		lenDelegates := len(delegates)

		wallet1, _ := getWallet(t, configPath)
		wallet1Balance, _ := getBalanceForWalletFromAPI(wallet1.ClientID)

		fmt.Println("wallet 1 balance : ", wallet1Balance)

		require.Equal(t, expectedWallet1Balance, wallet1Balance, "wallet 1 balance is not as expected")

		// stake 1 token to all the blobbers
		for _, blobber := range blobbersList {
			_, err := executeFaucetWithTokens(t, configPath, 9)
			if err != nil {
				return
			}

			expectedWallet1Balance += 8e10

			_, err = stakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
			if err != nil {
				return
			}

			wallet1Balance, _ = getBalanceForWalletFromAPI(wallet1.ClientID)
			require.Equal(t, expectedWallet1Balance, wallet1Balance, "wallet 1 balance is not as expected")
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew := len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		// create an allocation of capacity
		output, err := executeFaucetWithTokens(t, configPath, 9)
		if err != nil {
			fmt.Println(err)
			return
		}

		allocationId := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   32212254716,
			"tokens": 20,
			"data":   3,
			"parity": 3,
			"lock":   9,
		})

		fmt.Println(allocationId)

		// check total offers new value and compare
		output, _ = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobber.Id}))

		err = json.Unmarshal([]byte(output[0]), &blInfo)
		require.Nil(t, err, "error unmarshalling blobber info")

		totalOffersNew := blInfo.TotalOffers
		require.Equal(t, totalOffersNew, totalOffers+9999999999, "Total Offers should Increase")

		require.Greater(t, totalOffersNew, totalOffers, "total offers should be greater")

		newStakeWallet := "new_stake_wallet"

		_, err = registerWalletForName(t, configPath, newStakeWallet)
		if err != nil {
			return
		}

		wallet2, _ := getWalletForName(t, configPath, newStakeWallet)
		wallet2Balance, _ := getBalanceForWalletFromAPI(wallet2.ClientID)

		require.Equal(t, expectedWallet2Balance, wallet2Balance, "wallet 2 balance is not as expected")

		// stake 1 more token to blobbers
		for _, blobber := range blobbersList {
			_, err := executeFaucetWithTokensForWallet(t, newStakeWallet, configPath, 9)
			if err != nil {
				return
			}

			expectedWallet2Balance += 8e10

			_, err = stakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id, "tokens": 1}), true)
			if err != nil {
				return
			}

			wallet2Balance, _ = getBalanceForWalletFromAPI(wallet2.ClientID)
			require.Equal(t, expectedWallet2Balance, wallet2Balance, "wallet 2 balance is not as expected")
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates+1, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		wallet2Balance, _ = getBalanceForWalletFromAPI(wallet2.ClientID)
		require.Equal(t, expectedWallet2Balance, wallet2Balance, "wallet 2 balance is not as expected")

		// Try to unstake tokens from the blobbers
		for _, blobber := range blobbersList {
			output, err := unstakeTokensForWallet(t, configPath, newStakeWallet, createParams(map[string]interface{}{"blobber_id": blobber.Id}))

			expectedWallet2Balance += 1e10

			require.Nil(t, err, "error should not be there")
			fmt.Println(output)

			wallet2Balance, _ = getBalanceForWalletFromAPI(wallet2.ClientID)
			require.Equal(t, expectedWallet2Balance, wallet2Balance, "wallet 2 balance is not as expected")
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "delegates should be greater")

		lenDelegates = lenDelegatesNew

		wallet1Balance, _ = getBalanceForWalletFromAPI(wallet1.ClientID)
		fmt.Println(expectedWallet1Balance, wallet1Balance)
		expectedWallet1Balance = wallet1Balance
		//require.Equal(t, expectedWallet1Balance, wallet1Balance, "wallet 1 balance is not as expected")

		// Try to unstake tokens from the blobbers
		for _, blobber := range blobbersList {
			output, err := unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
			fmt.Println(output, err)

			wallet1Balance, _ = getBalanceForWalletFromAPI(wallet1.ClientID)
			fmt.Println(expectedWallet1Balance, wallet1Balance)
			//require.Equal(t, expectedWallet1Balance, wallet1Balance, "wallet 1 balance is not as expected")
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew, lenDelegates, "delegates should be equal")

		lenDelegates = lenDelegatesNew

		// Cancel the allocation
		output, err = cancelAllocation(t, configPath, allocationId, true)
		require.Nil(t, err, "error cancelling allocation")
		fmt.Println(output)

		// Try to unstake tokens from the blobbers

		wallet1Balance, _ = getBalanceForWalletFromAPI(wallet1.ClientID)
		fmt.Println(expectedWallet1Balance, wallet1Balance)
		expectedWallet1Balance = wallet1Balance
		//require.Equal(t, expectedWallet1Balance, wallet1Balance, "wallet 1 balance is not as expected")

		for _, blobber := range blobbersList {
			_, err := unstakeTokens(t, configPath, createParams(map[string]interface{}{"blobber_id": blobber.Id}))
			require.Nil(t, err, "error should not be there")

			expectedWallet1Balance += 1e10

			wallet1Balance, _ = getBalanceForWalletFromAPI(wallet1.ClientID)
			fmt.Println("wallet1Balance", wallet1Balance)
		}

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate
		lenDelegatesNew = len(delegates)

		require.Equal(t, lenDelegatesNew+1, lenDelegates, "delegates should be greater")
	})
}

type BlobberInfo struct {
	Id    string `json:"id"`
	Url   string `json:"url"`
	Terms struct {
		ReadPrice        int     `json:"read_price"`
		WritePrice       int     `json:"write_price"`
		MinLockDemand    float64 `json:"min_lock_demand"`
		MaxOfferDuration int64   `json:"max_offer_duration"`
	} `json:"terms"`
	Capacity          int64 `json:"capacity"`
	Allocated         int   `json:"allocated"`
	LastHealthCheck   int   `json:"last_health_check"`
	StakePoolSettings struct {
		DelegateWallet string  `json:"delegate_wallet"`
		MinStake       int64   `json:"min_stake"`
		MaxStake       int64   `json:"max_stake"`
		NumDelegates   int     `json:"num_delegates"`
		ServiceCharge  float64 `json:"service_charge"`
	} `json:"stake_pool_settings"`
	TotalStake               int64 `json:"total_stake"`
	UsedAllocation           int   `json:"used_allocation"`
	TotalOffers              int   `json:"total_offers"`
	TotalServiceCharge       int   `json:"total_service_charge"`
	UncollectedServiceCharge int   `json:"uncollected_service_charge"`
	IsKilled                 bool  `json:"is_killed"`
	IsShutdown               bool  `json:"is_shutdown"`
}

func getBalanceForWalletFromAPI(clientID string) (int64, error) {
	url := "https://test2.zus.network/sharder01/v1/client/get/balance?client_id=" + clientID
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response WalletBalanceAPIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}

	return response.Balance, nil
}

type WalletBalanceAPIResponse struct {
	Txn     string `json:"txn"`
	Round   int    `json:"round"`
	Balance int64  `json:"balance"`
	Nonce   int    `json:"nonce"`
}
