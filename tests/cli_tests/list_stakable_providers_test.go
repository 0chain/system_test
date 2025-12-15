package cli_tests

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/tokenomics_tests/utils"
	"github.com/stretchr/testify/require"
)

func TestGetStakableProviders(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("get stakable miners should work", func(t *test.SystemTest) {
		createWallet(t)

		// Get a stakable miner from the list
		stakableMinersBefore := getStakableMinersList(t)
		require.NotEmpty(t, stakableMinersBefore.Nodes, "No stakable miners found")

		// Try to use miner01ID if it exists in the list, otherwise use the first available miner
		selectedMinerID := stakableMinersBefore.Nodes[0].ID
		hasMiner01 := false
		for _, minerNode := range stakableMinersBefore.Nodes {
			if minerNode.ID == miner01ID {
				selectedMinerID = miner01ID
				hasMiner01 = true
				break
			}
		}
		log.Printf("Selected miner ID: %s (miner01ID found: %v)", selectedMinerID, hasMiner01)

		// count number of delegates - use test wallet for miner info
		output, err := minerInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"id": selectedMinerID,
		}), escapedTestName(t), true)
		require.Nilf(t, err, "error fetching miner info for miner %s: %v", selectedMinerID, err)
		require.Len(t, output, 1)
		var minerInfo1 model.Node
		err = json.Unmarshal([]byte(output[0]), &minerInfo1)
		require.Nilf(t, err, "error unmarshalling miner info: %v", err)
		delegateCnt := len(minerInfo1.Pools)
		log.Printf("minerInfo: %v", minerInfo1)
		log.Printf("num delegates: %d", delegateCnt)

		// Only update num_delegates if we have miner01ID and the delegate wallet
		if hasMiner01 {
			if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err == nil {
				// update num_delegates to delegateCnt + 1
				output, err = minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
					"id":            selectedMinerID,
					"num_delegates": delegateCnt + 1,
					"sharder":       false,
				}), true)
				require.Nilf(t, err, "error updating num_delegates for miner: %v", err)
				require.Len(t, output, 2)
				require.Equal(t, "settings updated", output[0])
				require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
				t.Cleanup(func() {
					// reset miner settings
					log.Printf("reset miner settings called")
					output, err = minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
						"id":            selectedMinerID,
						"num_delegates": minerInfo1.Settings.MaxNumDelegates,
						"sharder":       false,
					}), true)
					if err != nil {
						log.Printf("Warning: error reverting miner settings during cleanup: %v", err)
					}
				})
			} else {
				log.Printf("Delegate wallet not found, skipping miner settings update")
			}
		} else {
			log.Printf("miner01ID not found in stakable miners, skipping miner settings update")
		}

		// assert selectedMinerID is present in stakable miners
		hasSelectedMiner := false
		for _, minerNode := range stakableMinersBefore.Nodes {
			if minerNode.ID == selectedMinerID {
				hasSelectedMiner = true
				break
			}
		}
		require.True(t, hasSelectedMiner, "selected miner should be found in miners list")

		// Stake tokens on this miner
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
			"tokens":   1,
		}), true)
		require.Nilf(t, err, "err staking tokens on miner: %v", err)
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		t.Cleanup(func() {
			// Unstake the tokens
			log.Printf("unstake tokens called")
			output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"miner_id": selectedMinerID,
			}), true)
			if err != nil {
				log.Printf("Warning: error in unstake tokens during cleanup: %v", err)
			}
		})

		// assert selectedMinerID is not present in the stakable miners
		stakableMinersAfter := getStakableMinersList(t)
		hasSelectedMiner = false
		for _, minerNode := range stakableMinersAfter.Nodes {
			if minerNode.ID == selectedMinerID {
				hasSelectedMiner = true
				break
			}
		}
		require.False(t, hasSelectedMiner, "selected miner should NOT be found in miners list")
		require.Equal(t, len(stakableMinersAfter.Nodes), len(stakableMinersBefore.Nodes)-1, "stakableMinersAfter should be one less than stakableMinersBefore")
	})

	t.RunSequentially("get stakable sharders should work", func(t *test.SystemTest) {
		createWallet(t)

		// count number of delegates
		output, err := minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder01ID,
		}), true)
		require.Nilf(t, err, "error fetching sharder info: %v", err)
		require.Len(t, output, 1)
		var sharderInfo model.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nilf(t, err, "error unmarshalling sharder info: %v", err)
		delegateCnt := len(sharderInfo.Pools)
		log.Printf("sharderInfo: %v", sharderInfo)
		log.Printf("num delegates: %d", delegateCnt)

		// update num_delegates to delegateCnt + 1
		output, err = minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            sharder01ID,
			"num_delegates": delegateCnt + 1,
			"sharder":       true,
		}), true)
		require.Nilf(t, err, "error updating num_delegates for sharder01ID: %v", err)
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Cleanup(func() {
			// reset sharder settings
			output, err = minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            sharder01ID,
				"num_delegates": sharderInfo.Settings.MaxNumDelegates,
				"sharder":       true,
			}), true)
			require.Nilf(t, err, "error reverting sharder settings during cleanup: %v", err)
			require.Len(t, output, 2)
			require.Equal(t, "settings updated", output[0])
			require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		})

		// assert sharder01ID is present in stakable sharders
		stakableShardersBefore := getStakableSharderList(t)
		hasSharder01 := false
		for _, sharderNode := range stakableShardersBefore {
			if sharderNode.ID == sharder01ID {
				hasSharder01 = true
				break
			}
		}
		require.True(t, hasSharder01, "sharder01ID is not found in sharders list")

		// Stake tokens on this miner
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
			"tokens":     1,
		}), true)
		require.Nilf(t, err, "err staking tokens on sharder: %v", err)
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		t.Cleanup(func() {
			// Unstake the tokens
			output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"sharder_id": sharder01ID,
			}), true)
			require.Nilf(t, err, "error in unstake tokens during cleanup: %v", err)
		})

		// assert sharder01ID is not present in the stakable sharders
		stakableShardersAfter := getStakableSharderList(t)
		hasSharder01 = false
		for _, sharderNode := range stakableShardersAfter {
			if sharderNode.ID == sharder01ID {
				hasSharder01 = true
				break
			}
		}
		require.False(t, hasSharder01, "sharder01ID should NOT be present in sharders list")
		require.Equal(t, len(stakableShardersAfter), len(stakableShardersBefore)-1, "stakableShardersAfter should be one less than stakableShardersBefore")
	})

	t.RunSequentially("get stakable blobbers should work", func(t *test.SystemTest) {
		createWallet(t)

		stakableBlobbersBefore := getStakableBlobberList(t)
		blobberNode := stakableBlobbersBefore[0]

		// count number of delegates
		output, err := utils.StakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobberNode.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Error fetching stake pool info", strings.Join(output, "\n"))
		stakePoolInfo := model.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolInfo)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		delegateCnt := len(stakePoolInfo.Delegate)
		log.Printf("blobber stakePoolInfo: %v", stakePoolInfo)
		log.Printf("num delegates: %d", delegateCnt)

		// update num_delegates to delegateCnt + 1
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobberNode.Id,
			"num_delegates": delegateCnt + 1,
		}))
		require.Nil(t, err, "error updating num_delegates for blobber")
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])
		t.Cleanup(func() {
			// reset blobber settings
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobberNode.Id,
				"num_delegates": blobberNode.StakePoolSettings.MaxNumDelegates,
			}))
			require.Nilf(t, err, "error reverting blobber settings during cleanup: %v", err)
			require.Len(t, output, 1)
			require.Equal(t, "blobber settings updated successfully", output[0])
		})

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobberNode.Id,
			"tokens":     1.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])
		t.Cleanup(func() {
			// Unstake the tokens
			output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
				"blobber_id": blobberNode.Id,
			}), true)
			require.Nilf(t, err, "error in unstake tokens during cleanup: %v", err)
		})

		// assert blobberNode is not present in the stakable blobbers
		stakableBlobbersAfter := getStakableBlobberList(t)
		hasBlobberNode := false
		for _, blobber := range stakableBlobbersAfter {
			if blobber.Id == blobberNode.Id {
				hasBlobberNode = true
				break
			}
		}
		require.Falsef(t, hasBlobberNode, "staked blobber should NOT be present in blobbers list")
		require.Equal(t, len(stakableBlobbersAfter), len(stakableBlobbersBefore)-1, "stakableBlobbersAfter should be one less than stakableBlobbersBefore")
	})

	t.RunSequentially("get stakable validators should work", func(t *test.SystemTest) {
		createWallet(t)

		stakableValidatorsBefore := getStakableValidatorList(t)
		validatorNode := stakableValidatorsBefore[0]

		// count number of delegates
		output, err := utils.StakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"validator_id": validatorNode.ID,
			"json":         "",
		}))
		require.Nilf(t, err, "error fetching stake pool info: %v", err)
		require.Len(t, output, 1)
		stakePoolInfo := model.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolInfo)
		require.Nilf(t, err, "error unmarshalling stake pool info: %v", err)
		delegateCnt := len(stakePoolInfo.Delegate)
		log.Printf("validator stakePoolInfo: %v", stakePoolInfo)
		log.Printf("num delegates: %d", delegateCnt)

		// update num_delegates to delegateCnt + 1
		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{
			"validator_id":  validatorNode.ID,
			"num_delegates": delegateCnt + 1,
		}))
		require.Nilf(t, err, "error updating num_delegates: %v", err)
		require.Len(t, output, 1)
		t.Cleanup(func() {
			output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{
				"validator_id":  validatorNode.ID,
				"num_delegates": validatorNode.NumDelegates,
			}))
			require.Nilf(t, err, "error updating num_delegates during cleanup: %v", err)
			require.Len(t, output, 1)
		})

		// Stake tokens against this validator
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"validator_id": validatorNode.ID,
			"tokens":       1.0,
		}), true)
		require.Nilf(t, err, "error staking tokens: %v", err)
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])
		t.Cleanup(func() {
			// Unstake the tokens
			output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
				"validator_id": validatorNode.ID,
			}), true)
			require.Nilf(t, err, "error in unstake tokens during cleanup: %v", err)
		})

		// assert validatorNode is not present in the stakable validators
		stakableValidatorsAfter := getStakableValidatorList(t)
		hasValidatorNode := false
		for _, validator := range stakableValidatorsAfter {
			if validator.ID == validatorNode.ID {
				hasValidatorNode = true
				break
			}
		}
		require.Falsef(t, hasValidatorNode, "staked validator should NOT be present in validators list")
		require.Equal(t, len(stakableValidatorsAfter), len(stakableValidatorsBefore)-1, "stakableValidatorsAfter should be one less than stakableValidatorsBefore")
	})
}

func getStakableMinersList(t *test.SystemTest) *model.NodeList {
	// Get miner list.
	output, err := listMiners(t, configPath, createParams(map[string]interface{}{
		"stakable": true,
		"json":     "",
	}))
	require.Nil(t, err, "get stakable miners failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0, "Expected output to have length of at least 1")

	var miners model.NodeList
	log.Printf("json miners: %s", output[len(output)-1])
	err = json.Unmarshal([]byte(output[len(output)-1]), &miners)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", output[len(output)-1], err)
	require.NotEmpty(t, miners.Nodes, "No miners found: %v", strings.Join(output, "\n"))
	return &miners
}

func getStakableSharderList(t *test.SystemTest) []model.Node {
	// Get sharder list.
	output, err := listStakableShardersCommand(t, configPath)
	require.Nil(t, err, "get stakable sharders failed", strings.Join(output, ""))
	require.Greater(t, len(output), 0, "Expected output to have length of at least 1")

	var sharders []model.Node
	log.Printf("json sharders: %s", strings.Join(output, ""))
	err = json.Unmarshal([]byte(strings.Join(output, "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, ""), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output, ""))
	return sharders
}

func listStakableShardersCommand(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	t.Logf("list stakable sharder nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --active --stakable --json --silent --all --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func getStakableBlobberList(t *test.SystemTest) []model.BlobberInfo {
	// Get blobber list
	output, err := utils.ListBlobbers(t, configPath, createParams(map[string]interface{}{
		"stakable": true,
		"json":     "",
	}))
	require.Nilf(t, err, "error listing blobbers: %v", err)
	require.Len(t, output, 1)

	blobbers := []model.BlobberInfo{}
	err = json.Unmarshal([]byte(output[0]), &blobbers)
	require.Nilf(t, err, "error unmarshalling blobber list: %v", err)
	require.NotEmpty(t, blobbers, "No blobbers found in blobber list")
	return blobbers
}

func getStakableValidatorList(t *test.SystemTest) []model.Validator {
	output, err := utils.ListValidators(t, configPath, createParams(map[string]interface{}{
		"stakable": true,
		"json":     "",
	}))
	require.Nilf(t, err, "error listing validators: %v", err)
	require.Len(t, output, 1)

	var validators []model.Validator
	err = json.Unmarshal([]byte(output[0]), &validators)
	require.Nilf(t, err, "error unmarshalling validators list: %v", err)
	require.NotEmpty(t, validators, "No validators found in validators list")
	return validators
}

func minerInfoForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("Fetching miner node info...")
	return cliutil.RunCommand(t, fmt.Sprintf("./zwallet mn-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}
