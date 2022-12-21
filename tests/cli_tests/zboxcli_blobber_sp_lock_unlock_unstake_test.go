package cli_tests

import (
	"encoding/json"
	"fmt"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestBlobberStakePoolLockUnlock(testSetup *testing.T) { // nolint cyclomatic complexity 48
	t := test.NewSystemTest(testSetup)

	t.RunWithTimeout("test case 1", 4*time.Minute, func(t *test.SystemTest) {
		fileWalletMap := make(map[string]string)

		files, err := os.ReadDir("./config/")
		if err != nil {
			t.Log(err)
		}
		for _, file := range files {
			var content map[string]interface{}

			temp := strings.Split(file.Name(), ".")

			if len(temp) > 1 && temp[1] == "json" {
				data, err := os.ReadFile("./config/" + file.Name())
				if err != nil {
					t.Log(err)
				}
				err = json.Unmarshal([]byte(data), &content)
				fileWalletMap[file.Name()] = content["client_id"].(string)
			}
		}

		for k, v := range fileWalletMap {
			fmt.Printf("%s -- %s\n", k, v)
		}

		data, err := os.ReadFile("ignore")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(data))

		data, err = os.ReadFile("./config/wallets/blobber_owner_wallet.json")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(data))

		out, err := exec.Command("bash", "-c", "echo $HOME").Output()
		if err != nil {
			panic(err)
		}
		t.Logf("%s\n", out)
		myString := string(out[:(len(string(out)) - 1)])
		files1, err := os.ReadDir(myString)

		if err != nil {
			panic(err)
		}

		for _, file := range files1 {
			t.Log(file.Name())
		}
		myString = string(out[:(len(string(out)) - 1)])
		myString = filepath.Clean(filepath.Join(myString, ".."))
		t.Log(myString)
		files, err = os.ReadDir(myString)

		if err != nil {
			panic(err)
		}
		for _, file := range files {
			t.Log(file.Name())
		}

		out, err = exec.Command("bash", "-c", "pwd").Output()
		if err != nil {
			panic(err)
		}
		t.Logf("%s\n", out)
		myString = string(out[:(len(string(out)) - 1)])
		files, err = os.ReadDir(myString)
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			t.Log(file.Name())
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Registering wallet failed", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "Failed executing faucet", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Failed listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		t.Log(output)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Failed unmarshalling json", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found")
		t.Log(blobbers)

		blobber := blobbers[time.Now().Unix()%int64(len(blobbers)-1)]
		t.Log(blobber.Id)

		/*
			cleanupFunc := stakeAndUnstakePreExistingSP(t, "../ignore", blobber.Id, configPath)

			t.Cleanup(func() {
				cleanupFunc()
			})
		*/

		// Wallet balance should decrease by locked amount
		output, err = getBalance(t, configPath)
		t.Logf("The first balance: %s", output)
		require.Regexp(t, regexp.MustCompile(`Balance: 9.000 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		var spInfoFirst map[string]interface{}
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		t.Log(output)
		err = json.Unmarshal([]byte(output[0]), &spInfoFirst)
		t.Log(int64(spInfoFirst["stake_total"].(float64) / 1e10))

		//Lock tokens
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     4.0,
		}), true)
		require.Nil(t, err, "Failed staking tokens", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		var stakePoolFirst = climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePoolFirst)
		//		require.Len(t, output, 1)
		//		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])
		t.Log(output[0])

		// Use sp-info to check the staked tokens in blobber's stake pool
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		var spi = climodel.StakePoolInfo{}
		_ = json.Unmarshal([]byte(output[0]), &spi)
		t.Log(output[0])
		for _, aa := range spi.Delegate {
			fmt.Println(aa.DelegateID)
		}

		//	require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		//	require.Len(t, output, 1)
		//	require.Regexp(t, regexp.MustCompile(`Balance: 2.800 ZCN \(\d*\.?\d+ USD\)$`), output[0])

		name := cliutils.RandomAlphaNumericString(10)
		options := map[string]interface{}{
			"lock": 3.0,
			"name": name,
		}
		output, err = createNewAllocation(t, configPath, createParams(options))
		t.Log(output)
		require.Nil(t, err, "Failed creating new allocation", strings.Join(output, "\n"))
		//		require.True(t, len(output) > 0, "expected output length be at least 1")
		//		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))
		allocationID := strings.Fields(output[0])[2]
		t.Log(allocationID)

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		//Lock tokens
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     2.0,
		}), true)
		require.Nil(t, err, "Failed staking tokens", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		// Use sp-info to check the staked tokens in blobber's stake pool
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		t.Log(output)

		var spINFO climodel.StakePoolInfo
		err = json.Unmarshal([]byte(output[0]), &spINFO)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		//	require.NotEmpty(t, spInfoAfterStake)
		//	t.Log(int64(spInfoAfterStake["stake_total"].(float64)))
		//	require.Greater(t, int64(spInfoAfterStake["stake_total"].(float64)), int64(spInfoFirst["stake_total"].(float64)), "Total Stake Is Not Increased!")

		//unlock tokens for non offered tokens
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}))
		t.Log(output)
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))
		//	require.Len(t, output, 1)
		//	require.Equal(t, "tokens unlocked, pool deleted", output[0])

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		//	unlock tokens for offered tokens should fail
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}))
		t.Log(output[0])
		require.NotNil(t, err, strings.Join(output, "\n"))

		//	cancel allocation
		output, err = cancelAllocation(t, configPath, allocationID, true)
		require.Nil(t, err, "Error cancelling allocation", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		t.Log(output)

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		//	unlock tokens for offered tokens should fail
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}))
		t.Log(output[0])
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)
		t.Logf("Balance: %v", output)

		// Use sp-info to check the staked tokens in blobber's stake pool
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		t.Log(output)
	})
}

func stakeAndUnstakePreExistingSP(t *test.SystemTest, wallet, blobberID, cliconfigpath string) func() {
	t.Log("Unstake pre-existing stake-pools...")
	output, err := stakePoolInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id": blobberID,
		"json":       "",
	}))
	require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
	t.Log(output)

	var spINFO climodel.StakePoolInfo
	var delegateInfo climodel.StakePoolDelegatePoolInfo
	err = json.Unmarshal([]byte(output[0]), &spINFO)

	var backupOfStakes int64 = 0

	for _, delegateInfo = range spINFO.Delegate {
		backupOfStakes += delegateInfo.Balance / 1e10
	}

	output, err = unstakeTokensForWallet(t, cliconfigpath, "../ignore", createParams(map[string]interface{}{
		"blobber_id": blobberID,
	}))

	return func() {
		fmt.Println("Re-stake unstaked pre-existed stake-pools...")
		output, err = stakeTokensForWallet(t, configPath, wallet, createParams(map[string]interface{}{
			"blobber_id": blobberID,
			"tokens":     backupOfStakes,
		}), true)

		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobberID,
			"json":       "",
		}))
		fmt.Println(output)
	}
}

func unstakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func stakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string, retry bool) ([]string, error) {
	t.Log("Staking tokens...")
	cmd := fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
