package cli_tests

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestVestingPoolTokenAccounting(t *testing.T) {
	t.Parallel()

	t.Run("Vesting pool with one destination should move some balance to pending which should be unlockable", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		startTime := time.Now().Add(1 * time.Second).Unix()

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":          targetWallet.ClientID + ":2",
			"lock":       2,
			"duration":   "2m",
			"start_time": startTime,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully: [a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		cliutils.Wait(t, 30*time.Second)

		// Get vp-info and current time
		output, err = vestingPoolInfo(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true)
		currTime := time.Now().Unix()
		require.Nil(t, err, "error fetching pool info")
		require.Len(t, output, 18, "expected output of length 18")
		ratio := (float64(currTime) - float64(startTime)) / 120 // 120 is duration
		expectedTransferAmount := 2 * ratio                     // 0.1 is destination amount
		actualTransferAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[15]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		unit := regexp.MustCompile("[um]?ZCN").FindString(output[15])
		actualTransferAmount = unitToZCN(actualTransferAmount, unit)
		require.GreaterOrEqualf(t, actualTransferAmount, expectedTransferAmount,
			"transferred amount [%v] should have been greater than or equal to expected transferred amount [%v]", actualTransferAmount, expectedTransferAmount)

		// Target wallet should be able to unlock tokens from vesting pool
		output, err = vestingPoolUnlockForWallet(t, configPath, createParams(map[string]interface{}{
			"pool_id": poolId,
		}), true, targetWalletName)
		require.Nil(t, err, "error unlocking tokens from vesting pool by target wallet")
		require.Len(t, output, 1)
		require.Equal(t, "Tokens unlocked successfully.", output[0])

		output, err = getBalanceForWallet(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching balance for target wallet")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d+\.?\d* [um]?ZCN \(\d+\.?\d* USD\)`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?ZCN`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		newBalanceInZCN := unitToZCN(newBalanceValue, strings.Fields(newBalance)[1])
		require.GreaterOrEqualf(t, newBalanceInZCN, actualTransferAmount,
			"amount in wallet after unlock should be greater or equal to transferred amount")
	})
}
