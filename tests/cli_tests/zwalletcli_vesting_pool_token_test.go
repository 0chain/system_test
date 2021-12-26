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

	t.Run("Vesting pool with one destination should move tokens from balance to pending", func(t *testing.T) {
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
		ratio := (float64(currTime) - float64(startTime))/120 // 120 is duration
		expectedTransferAmount := 2 * ratio                 // 0.1 is destination amount
		actualTransferAmount, err := strconv.ParseFloat(regexp.MustCompile(`\d+\.?\d*`).FindString(output[15]), 64)
		require.Nil(t, err, "error parsing float from vp-info")
		unit := regexp.MustCompile("[um]?ZCN").FindString(output[15])
		actualTransferAmount = unitToZCN(actualTransferAmount, unit)
		require.InEpsilon(t, expectedTransferAmount, actualTransferAmount, 0.05)
	})
}
