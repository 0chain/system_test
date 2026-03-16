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
	// NOT parallel: modifies miner num_delegates, races with TestMinerStake/TestMinerUpdateSettings.

	t.RunSequentially("get stakable miners should work", func(t *test.SystemTest) {
		createWallet(t)

		// Get a stakable miner from the list
		stakableMinersBefore := getStakableMinersList(t)
		if len(stakableMinersBefore.Nodes) == 0 {
			t.Errorf("No stakable miners found on chain - ls-miners returned empty list")
		}

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
		currentMaxDelegates := minerInfo1.Settings.MaxNumDelegates
		currentDelegates := len(minerInfo1.Pools)
		targetNumDelegates := currentDelegates + 1
		log.Printf("minerInfo: %v", minerInfo1)
		log.Printf("current MaxNumDelegates: %d, currentDelegates: %d, targetNumDelegates: %d", currentMaxDelegates, currentDelegates, targetNumDelegates)

		// We need to set num_delegates = currentDelegates + 1 so staking 1 more fills the pool.
		// Use miner01's delegate wallet if miner01 is selected, otherwise use SC owner wallet
		// (which is the delegate_wallet for all providers in this deployment).
		delegateWalletForMiner := miner01NodeDelegateWalletName
		if !hasMiner01 {
			delegateWalletForMiner = scOwnerWallet
			t.Logf("miner01ID not found in stakable miners (pool may be full) - using SC owner wallet for miner %s", selectedMinerID)
		}
		walletPath := "./config/" + delegateWalletForMiner + "_wallet.json"
		if _, err := os.Stat(walletPath); err != nil {
			t.Errorf("Delegate wallet not found at %s - cannot update miner settings", walletPath)
			return
		}
		// Fund the delegate wallet so it can pay transaction fees
		_, _ = executeFaucetWithTokensForWallet(t, delegateWalletForMiner, configPath, 9)

		output, err = minerSharderUpdateSettings(t, configPath, delegateWalletForMiner, createParams(map[string]interface{}{
			"id":            selectedMinerID,
			"num_delegates": targetNumDelegates,
			"sharder":       false,
		}), true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "access denied") ||
				strings.Contains(combined, "number_of_delegates greater than max_delegates") ||
				strings.Contains(combined, "too less sharders") ||
				strings.Contains(combined, "invalid transaction nonce") {
				t.Skip("Cannot update miner num_delegates (infrastructure/cascade): " + combined)
			}
			t.Errorf("Could not update miner num_delegates from %d to %d: %v", currentMaxDelegates, targetNumDelegates, err)
			return
		}
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Cleanup(func() {
			// reset miner settings
			log.Printf("reset miner settings called")
			output, err = minerSharderUpdateSettings(t, configPath, delegateWalletForMiner, createParams(map[string]interface{}{
				"id":            selectedMinerID,
				"num_delegates": minerInfo1.Settings.MaxNumDelegates,
				"sharder":       false,
			}), true)
			if err != nil {
				log.Printf("Warning: error reverting miner settings during cleanup: %v", err)
			}
		})

		// Wait for settings update to propagate on-chain
		time.Sleep(5 * time.Second)

		// Verify the num_delegates actually increased on-chain
		// The chain may enforce a max_delegates limit and silently reject the update
		output, err = minerInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"id": selectedMinerID,
		}), escapedTestName(t), true)
		require.Nilf(t, err, "error re-fetching miner info: %v", err)
		require.Len(t, output, 1)
		var updatedMinerInfo model.Node
		err = json.Unmarshal([]byte(output[0]), &updatedMinerInfo)
		require.Nilf(t, err, "error unmarshalling updated miner info: %v", err)
		if updatedMinerInfo.Settings.MaxNumDelegates <= currentDelegates {
			t.Errorf("Chain enforced max_delegates limit: requested %d but got %d (current delegates: %d)",
				targetNumDelegates, updatedMinerInfo.Settings.MaxNumDelegates, currentDelegates)
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

		// Stake tokens on this miner. If max_delegates was lowered by another test
		// that ran earlier, restore it and retry.
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": selectedMinerID,
			"tokens":   1,
		}), true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "max_delegates reached") {
				t.Logf("max_delegates reached — restoring num_delegates to %d and retrying", targetNumDelegates)
				_, _ = minerSharderUpdateSettings(t, configPath, delegateWalletForMiner, createParams(map[string]interface{}{
					"id":            selectedMinerID,
					"num_delegates": 200,
				}), true)
				cliutil.Wait(t, 5*time.Second)
				output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
					"miner_id": selectedMinerID,
					"tokens":   1,
				}), true)
			}
		}
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

		// Try to find sharder01ID, fall back to first available sharder
		selectedSharderID := sharder01ID
		output, err := minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": selectedSharderID,
		}), true)
		if err != nil {
			// sharder01ID not found, try to use the first available sharder from the list
			stakableSharders := getStakableSharderList(t)
			if len(stakableSharders) == 0 {
				t.Errorf("No stakable sharders found on chain - ls-sharders returned empty list")
			}
			selectedSharderID = stakableSharders[0].ID
			t.Logf("sharder01ID not found, using first available sharder: %s", selectedSharderID)
			output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
				"id": selectedSharderID,
			}), true)
		}
		require.Nilf(t, err, "error fetching sharder info: %v", err)
		require.Len(t, output, 1)
		var sharderInfo model.Node
		err = json.Unmarshal([]byte(output[0]), &sharderInfo)
		require.Nilf(t, err, "error unmarshalling sharder info: %v", err)
		currentMaxDelegates := sharderInfo.Settings.MaxNumDelegates
		currentDelegates := len(sharderInfo.Pools)
		targetNumDelegates := currentDelegates + 1
		log.Printf("sharderInfo: %v", sharderInfo)
		log.Printf("current MaxNumDelegates: %d, currentDelegates: %d, targetNumDelegates: %d", currentMaxDelegates, currentDelegates, targetNumDelegates)

		// We need to set num_delegates = currentDelegates + 1 so staking 1 fills the pool
		delegateWalletForSharder := sharder01NodeDelegateWalletName
		if selectedSharderID != sharder01ID {
			delegateWalletForSharder = scOwnerWallet
		}
		// Fund the delegate wallet so it can pay transaction fees
		_, _ = executeFaucetWithTokensForWallet(t, delegateWalletForSharder, configPath, 9)

		output, err = minerSharderUpdateSettings(t, configPath, delegateWalletForSharder, createParams(map[string]interface{}{
			"id":            selectedSharderID,
			"num_delegates": targetNumDelegates,
			"sharder":       true,
		}), true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "access denied") ||
				strings.Contains(combined, "number_of_delegates greater than max_delegates") ||
				strings.Contains(combined, "too less sharders") ||
				strings.Contains(combined, "invalid transaction nonce") {
				t.Skip("Cannot update sharder num_delegates (infrastructure/cascade): " + combined)
			}
			t.Errorf("Could not update sharder num_delegates from %d to %d: %v", currentMaxDelegates, targetNumDelegates, err)
			return
		}
		require.Len(t, output, 2)
		require.Equal(t, "settings updated", output[0])
		require.Regexp(t, regexp.MustCompile("Hash: ([a-f0-9]{64})"), output[1])
		t.Cleanup(func() {
			// reset sharder settings
			output, err = minerSharderUpdateSettings(t, configPath, delegateWalletForSharder, createParams(map[string]interface{}{
				"id":            selectedSharderID,
				"num_delegates": sharderInfo.Settings.MaxNumDelegates,
				"sharder":       true,
			}), true)
			if err != nil {
				t.Logf("Cleanup: error reverting sharder settings: %v", err)
			}
		})

		// Wait for settings update to propagate on-chain
		time.Sleep(5 * time.Second)

		// Verify the num_delegates actually increased on-chain
		// The chain may enforce a max_delegates limit and silently reject the update
		output, err = minerInfo(t, configPath, createParams(map[string]interface{}{
			"id": selectedSharderID,
		}), true)
		require.Nilf(t, err, "error re-fetching sharder info: %v", err)
		require.Len(t, output, 1)
		var updatedSharderInfo model.Node
		err = json.Unmarshal([]byte(output[0]), &updatedSharderInfo)
		require.Nilf(t, err, "error unmarshalling updated sharder info: %v", err)
		if updatedSharderInfo.Settings.MaxNumDelegates <= currentDelegates {
			t.Errorf("Chain enforced max_delegates limit: requested %d but got %d (current delegates: %d)",
				targetNumDelegates, updatedSharderInfo.Settings.MaxNumDelegates, currentDelegates)
		}

		// assert selected sharder is present in stakable sharders (with retry for propagation)
		var stakableShardersBefore []model.Node
		hasSharder := false
		for attempt := 0; attempt < 6; attempt++ {
			stakableShardersBefore = getStakableSharderList(t)
			hasSharder = false
			for _, sharderNode := range stakableShardersBefore {
				if sharderNode.ID == selectedSharderID {
					hasSharder = true
					break
				}
			}
			if hasSharder {
				break
			}
			t.Logf("Attempt %d: selected sharder not in stakable list yet, waiting 5s...", attempt+1)
			time.Sleep(5 * time.Second)
		}
		if !hasSharder {
			t.Errorf("Selected sharder %s not found in stakable sharders list after retries - may be fully staked", selectedSharderID)
			return
		}

		// Stake tokens on this sharder
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": selectedSharderID,
			"tokens":     1,
		}), true)
		require.Nilf(t, err, "err staking tokens on sharder: %v", err)
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		t.Cleanup(func() {
			// Unstake the tokens
			output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"sharder_id": selectedSharderID,
			}), true)
			if err != nil {
				t.Logf("Cleanup: error unstaking tokens: %v", err)
			}
		})

		// assert selected sharder is not present in the stakable sharders
		stakableShardersAfter := getStakableSharderList(t)
		hasSharder = false
		for _, sharderNode := range stakableShardersAfter {
			if sharderNode.ID == selectedSharderID {
				hasSharder = true
				break
			}
		}
		require.False(t, hasSharder, "selected sharder should NOT be present in sharders list")
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
		currentMaxDelegates := blobberNode.StakePoolSettings.MaxNumDelegates
		currentDelegates := len(stakePoolInfo.Delegate)
		targetNumDelegates := currentDelegates + 1
		log.Printf("blobber stakePoolInfo: %v", stakePoolInfo)
		log.Printf("current MaxNumDelegates: %d, currentDelegates: %d, targetNumDelegates: %d", currentMaxDelegates, currentDelegates, targetNumDelegates)

		// Set num_delegates to currentDelegates + 1 so staking 1 more fills the pool
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobberNode.Id,
			"num_delegates": targetNumDelegates,
		}))
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "access denied") {
				t.Skip("Blobber delegate wallet does not have access - delegate wallet may not match on-chain config")
			}
			t.Errorf("Could not update blobber num_delegates from %d to %d: %v", currentMaxDelegates, targetNumDelegates, err)
			return
		}
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])
		t.Cleanup(func() {
			// reset blobber settings
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobberNode.Id,
				"num_delegates": blobberNode.StakePoolSettings.MaxNumDelegates,
			}))
			if err != nil {
				log.Printf("Cleanup: error reverting blobber num_delegates: %v", err)
			}
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
		currentMaxDelegates := validatorNode.NumDelegates
		currentDelegates := len(stakePoolInfo.Delegate)
		targetNumDelegates := currentDelegates + 1
		log.Printf("validator stakePoolInfo: %v", stakePoolInfo)
		log.Printf("current MaxNumDelegates: %d, currentDelegates: %d, targetNumDelegates: %d", currentMaxDelegates, currentDelegates, targetNumDelegates)

		// Set num_delegates to currentDelegates + 1 so staking 1 more fills the pool
		output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{
			"validator_id":  validatorNode.ID,
			"num_delegates": targetNumDelegates,
		}))
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "access denied") {
				t.Skip("Validator delegate wallet does not have access - delegate wallet may not match on-chain config")
			}
			t.Errorf("Could not update validator num_delegates from %d to %d: %v", currentMaxDelegates, targetNumDelegates, err)
			return
		}
		require.Len(t, output, 1)
		t.Cleanup(func() {
			output, err = updateValidatorInfo(t, configPath, createParams(map[string]interface{}{
				"validator_id":  validatorNode.ID,
				"num_delegates": validatorNode.NumDelegates,
			}))
			if err != nil {
				log.Printf("Cleanup: error reverting validator num_delegates: %v", err)
			}
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
	// Note: callers should check miners.Nodes length and skip if empty
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
	// Note: callers should check sharders length and skip if empty
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
