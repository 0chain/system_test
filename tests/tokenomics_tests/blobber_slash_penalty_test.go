package tokenomics_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestBlobberSlashPenalty(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Skip by default: this test kills a blobber on-chain which is permanent and
	// breaks other tests that need live blobbers for allocations.
	if os.Getenv("ENABLE_KILL_TESTS") == "" {
		t.Skip("Skipping slash penalty test - kills blobber on-chain (set ENABLE_KILL_TESTS=1 to run)")
	}

	var transientErr error
	t.TestSetup("set storage config to use time_unit as 1 month", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "720h",
		}, true)
		if err != nil && utils.IsTransientError(output, err) {
			transientErr = err
			return
		}
		require.Nil(t, err, strings.Join(output, "\n"))
	})
	if transientErr != nil {
		testSetup.Fatalf("Skipping test due to transient infrastructure error: %v", transientErr)
		return
	}

	t.Cleanup(func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "720h",
		}, true)
		if err != nil {
			t.Logf("Warning: failed to reset time_unit in cleanup: %v %s", err, strings.Join(output, "\n"))
		}
	})

	prevBlock := utils.GetLatestFinalizedBlock(t)

	t.Log("prevBlock", prevBlock)

	output, err := utils.CreateWallet(t, configPath)
	if err != nil && utils.IsTransientError(output, err) {
		testSetup.Fatal("Skipping test due to transient infrastructure error during wallet creation: ", err)
		return
	}
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of blobber list output")

	err = json.Unmarshal([]byte(output[len(output)-1]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	// zbox ls-blobbers --json does not return is_enterprise; query SC REST API directly.
	enterpriseBlobberIDs := make(map[string]bool)
	for _, eb := range utils.GetEnterpriseBlobbers(t) {
		enterpriseBlobberIDs[eb.ID] = true
	}

	var blobberListString []string
	for _, blobber := range blobberList {
		// Skip enterprise blobbers (CLI field unreliable; use SC REST API list)
		if blobber.IsEnterprise || enterpriseBlobberIDs[blobber.Id] {
			continue
		}
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.GreaterOrEqual(t, len(output), 1, "Expected at least 1 line of validator list output")

	err = json.Unmarshal([]byte(output[len(output)-1]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	// These tests use allocations with data=1, parity=1, requiring only 2 blobbers
	// and 2 validators. Skip gracefully if not enough are available.
	if len(blobberListString) < 2 {
		testSetup.Fatal("Need at least 2 blobbers for slash penalty tests, only found ", len(blobberListString))
		return
	}
	if len(validatorListString) < 2 {
		testSetup.Fatal("Need at least 2 validators for slash penalty tests, only found ", len(validatorListString))
		return
	}
	blobberListString = blobberListString[:2]
	validatorListString = validatorListString[:2]

	t.RunSequentiallyWithTimeout("Upload 10% of allocation and Kill blobber in the middle, One blobber should get approx double rewards than other", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1, defaultDelegateWalletSet())

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocation(t, configPath, map[string]interface{}{
			"size":   1 * GB,
			"tokens": 1,
			"data":   1,
			"parity": 1,
		})

		remotepath := "/dir/"
		filesize := 0.1 * GB
		filename := utils.GenerateRandomTestFileName(t)

		err = utils.CreateFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		output, err = utils.UploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationId,
			"remotepath": remotepath + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		// check allocation remaining time
		allocation := utils.GetAllocation(t, allocationId)
		remainingTime := allocation.ExpirationDate - time.Now().Unix()

		// sleep for half of the remaining time
		time.Sleep(time.Duration(remainingTime/3) * time.Second)

		// 2. Kill a blobber
		_, err = killBlobber(t, configPath, utils.CreateParams(map[string]interface{}{
			"id": blobberList[1].Id,
		}), true)
		require.Nil(t, err, "error killing blobber", strings.Join(output, "\n"))

		// 3. Sleep for the remaining time
		time.Sleep(time.Duration(remainingTime) * time.Second)

		allocation = utils.GetAllocation(t, allocationId)

		t.Log(allocation.MovedToChallenge)

		blobberRewards := getAllocationChallengeRewards(t, allocationId)

		t.Log(blobberRewards)

		blobber1Reward := blobberRewards[blobberList[0].Id].(float64)
		blobber2Reward := blobberRewards[blobberList[1].Id].(float64)

		t.Log(blobber1Reward, blobber2Reward)

		require.Greater(t, blobber1Reward/blobber2Reward, 2.0, "Killed blobber should get approx half the rewards than other")
	})
}

func killBlobber(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("kill blobber...")
	cmd := fmt.Sprintf("./zbox kill-blobber %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		params, scOwnerWallet, cliConfigFilename)

	t.Log(cmd)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
