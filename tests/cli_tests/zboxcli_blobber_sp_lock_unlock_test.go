package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberStakePoolLockUnlock(t *testing.T) {
	t.Parallel()

	t.Run("Lock tokens in blobber's stake pool should work", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

		// wallet, err := getWallet(t, configPath)
		// require.Nil(t, err, "Error getting wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.8,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		stakePoolFirstID := strings.Fields(output[0])[4]
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, pool id: ([a-f0-9]{64})"), output[0])

		// open allocation make sure to use most of the blobber stake?
		name := cliutils.RandomAlphaNumericString(10)

		options := map[string]interface{}{
			"lock": "0.7",
			"name": name,
		}
		output, err = createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))
		allocationID := strings.Fields(output[0])[4]

		// Wallet balance should decrease by locked amount
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 200.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		// Use sp-info to check the staked tokens in blobber's stake pool
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
		require.Regexp(t, regexp.MustCompile(`Balance: 800.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		//Lock more tokens in blobbers stake pool
		outputForSecondLock, err := stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.2,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(outputForSecondLock, "\n"))
		stakePoolSecondID := strings.Fields(output[0])[4]
		require.Len(t, outputForSecondLock, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, pool id: ([a-f0-9]{64})"), outputForSecondLock[0])

		// Use sp-info to check the staked tokens in blobber's stake pool
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 2)

		stakePool = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)
		require.Regexp(t, regexp.MustCompile(`Balance: 1000.00\d mZCN \(\d*\.?\d+ USD\)$`), output[0])

		//unlock tokens for non offered tokens
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"pool_id":    stakePoolSecondID,
		}))
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked, pool deleted", output[0])

		//unlock tokens for offered tokens should fail
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"pool_id":    stakePoolFirstID,
		}))
		require.Equal(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))

		//cancel allocation
		output, err = cancelAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}), true)
		require.Nil(t, err, "Error cancelling allocation")
		require.Len(t, output, 1)

		//unlock tokens should work
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"pool_id":    stakePoolFirstID,
		}))
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked, pool deleted", output[0])

	})

}
