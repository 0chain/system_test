package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/currency"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// Fixed
func TestBlobberCollectRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test collect reward with valid pool and blobber id should pass")

	t.Parallel()

	t.TestSetup("Create temp dir", func() {
		// Create a folder to keep all the generated files to be uploaded
		err := os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	t.Run("Test collect reward with valid pool and blobber id should pass", func(t *test.SystemTest) { // TODO slow
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		// Upload and download a file so blobber can accumulate rewards
		allocSize := int64(1 * GB)
		filesize := int64(256)
		remotepath := "/"

		// Use all blobbers
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   3,
			"parity": 1,
		})

		alloc := getAllocation(t, allocationID)
		blobberID := alloc.Blobbers[time.Now().Unix()%int64(len(alloc.Blobbers))].ID

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobberID,
			"tokens":     5.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  "tmp/",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobberID,
			"json":       "",
		}))
		require.Nil(t, err, "error getting stake pool info")
		require.Len(t, output, 1)
		stakePoolAfter := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolAfter)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePoolAfter)

		rewards := int64(0)
		for _, poolDelegateInfo := range stakePoolAfter.Delegate {
			if poolDelegateInfo.DelegateID == wallet.ClientID {
				rewards = poolDelegateInfo.TotalReward
				break
			}
		}
		require.Greater(t, rewards, int64(0))
		t.Logf("reward tokens: %v", rewards)

		balanceBefore := getBalanceFromSharders(t, wallet.ClientID)
		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   blobberID,
			"fee":           "0.15",
		}), true)
		require.NoError(t, err, output)

		feeTxn, err := currency.ParseZCN(0.15)
		require.NoError(t, err)
		balanceAfter := getBalanceFromSharders(t, wallet.ClientID)
		require.GreaterOrEqual(t, balanceAfter, balanceBefore-int64(feeTxn)+rewards)
	})

	t.Run("Test collect reward with invalid blobber id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "blobber",
			"provider_id":   "invalid-blobber-id",
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "collect_reward_failed")
	})

	t.Run("Test collect reward with invalid provider type should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

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
			"tokens":     0.5,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "invalid-provider",
			"provider_id":   blobber.Id,
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "provider type must be blobber or validator")
	})

	t.Run("Test collect reward with no provider id or type should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "missing tokens flag")
	})
}

func TestValidatorCollectRewards(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Test collect reward with valid pool and validator id should pass")

	t.Parallel()

	t.RunWithTimeout("Test collect reward with valid pool and validator id should pass", 600*time.Second, func(t *test.SystemTest) { // TODO slow
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		var validators []climodel.Validator
		output, err = listValidators(t, configPath, "--json")
		require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &validators)
		require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
		require.True(t, len(validators) > 0, "No validators found in validator list")

		// Pick a random blobber
		validator := validators[time.Now().Unix()%int64(len(validators))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
			"tokens":       1.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		balanceBefore := getBalanceFromSharders(t, wallet.ClientID)

		// Upload and download a file so blobber can accumulate rewards
		allocSize := 100 * MB
		filesize := 10 * MB
		remotepath := "/"

		// Use all blobbers
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"data":   4,
			"parity": 1,
		})

		filename := generateFileAndUpload(t, allocationID, remotepath, int64(filesize))

		// Delete the uploaded file, since we will be downloading it now
		err = os.Remove(filename)
		require.Nil(t, err)

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  os.TempDir() + string(os.PathSeparator),
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))

		cliutils.Wait(t, 30*time.Second)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
			"json":         "",
		}))
		require.Nil(t, err, "error getting stake pool info")
		require.Len(t, output, 1)
		stakePoolAfter := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolAfter)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePoolAfter)

		fmt.Println("stakePoolAfter", stakePoolAfter)

		rewards := int64(0)
		for _, poolDelegateInfo := range stakePoolAfter.Delegate {
			if poolDelegateInfo.DelegateID == wallet.ClientID {
				rewards = poolDelegateInfo.TotalReward
				fmt.Println("rewards", rewards)
				break
			}
		}
		// TODO: fix me - rewards should be greater than 0
		require.Greater(t, rewards, int64(0))

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "validator",
			"provider_id":   validator.ID,
		}), true)
		require.Nil(t, err, "Error collecting rewards", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "transferred reward tokens", output[0])

		balanceAfter := getBalanceFromSharders(t, wallet.ClientID)
		require.GreaterOrEqual(t, balanceAfter, balanceBefore+rewards) // greater or equal since more rewards can accumulate after we check stakepool
	})

	t.Run("Test collect reward with invalid validator id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		validators := []climodel.Validator{}
		output, err = listValidators(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &validators)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(validators) > 0, "No blobbers found in blobber list")

		// Pick a random validator
		validator := validators[time.Now().Unix()%int64(len(validators))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
			"tokens":       1.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "validator",
			"provider_id":   "invalid-validator-id",
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "collect_reward_failed")
	})

	t.Run("Test collect reward with invalid provider type should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		validators := []climodel.Validator{}
		output, err = listValidators(t, configPath, "--json")
		require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &validators)
		require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
		require.True(t, len(validators) > 0, "No validators found in validator list")

		// Pick a random blobber
		validator := validators[time.Now().Unix()%int64(len(validators))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validator.ID,
			"tokens":       1.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{
			"provider_type": "invalid-provider",
			"provider_id":   validator.ID,
		}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "provider type must be blobber or validator")
	})

	t.Run("Test collect reward with no provider id or type should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = collectRewards(t, configPath, createParams(map[string]interface{}{}), false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "missing tokens flag")
	})
}

func collectRewards(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("collecting rewards...")
	cmd := fmt.Sprintf("./zbox collect-reward %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
