package tokenomics_tests

import (
	"encoding/json"
	"fmt"
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

	t.TestSetup("set storage config to use time_unit as 10 minutes", func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "20m",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	t.Cleanup(func() {
		output, err := utils.UpdateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"time_unit": "1h",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
	})

	prevBlock := utils.GetLatestFinalizedBlock(t)

	t.Log("prevBlock", prevBlock)

	output, err := utils.CreateWallet(t, configPath)
	require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

	var blobberList []climodel.BlobberInfo
	output, err = utils.ListBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobberList)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobberList) > 0, "No blobbers found in blobber list")

	var blobberListString []string
	for _, blobber := range blobberList {
		blobberListString = append(blobberListString, blobber.Id)
	}

	var validatorList []climodel.Validator
	output, err = utils.ListValidators(t, configPath, "--json")
	require.Nil(t, err, "Error listing validators", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &validatorList)
	require.Nil(t, err, "Error unmarshalling validator list", strings.Join(output, "\n"))
	require.True(t, len(validatorList) > 0, "No validators found in validator list")

	var validatorListString []string
	for _, validator := range validatorList {
		validatorListString = append(validatorListString, validator.ID)
	}

	t.RunSequentiallyWithTimeout("Upload 10% of allocation and Kill blobber in the middle, One blobber should get approx double rewards than other", 1*time.Hour, func(t *test.SystemTest) {
		stakeTokensToBlobbersAndValidators(t, blobberListString, validatorListString, configPath, []float64{
			1, 1, 1, 1,
		}, 1)

		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// 1. Create an allocation with 1 data shard and 1 parity shard.
		allocationId := utils.SetupAllocationAndReadLock(t, configPath, map[string]interface{}{
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
