package utils

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	StatusCompletedCB         = "Status completed callback"
	FreeTokensIndividualLimit = 10.0
	FreeTokensTotalLimit      = 100.0
)

var (
	createAllocationRegex = regexp.MustCompile(`^Allocation created: (.+)$`)
	reAuthToken           = regexp.MustCompile(`^Auth token :(.*)$`)
)

func SetupAllocationAndReadLock(t *test.SystemTest, cliConfigFilename string, extraParam map[string]interface{}) string {
	tokens := float64(1)
	if tok, ok := extraParam["tokens"]; ok {
		token, err := strconv.ParseFloat(fmt.Sprintf("%v", tok), 64)
		require.Nil(t, err)
		tokens = token
	}

	allocationID := setupAllocation(t, cliConfigFilename, extraParam)

	// Lock half the tokens for read pool
	readPoolParams := CreateParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	output, err := ReadPoolLock(t, cliConfigFilename, readPoolParams, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	return allocationID
}

func ReadPoolLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return readPoolLockWithWallet(t, EscapedTestName(t), cliConfigFilename, params, retry)
}

func readPoolLockWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Locking read tokens...")
	cmd := fmt.Sprintf("./zbox rp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func setupAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return setupAllocationWithWallet(t, EscapedTestName(t), cliConfigFilename, extraParams...)
}

func setupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	faucetTokens := float64(2.0)
	// Then create new allocation
	options := map[string]interface{}{"expire": "1h", "size": "10000", "lock": "0.5"}

	// Add additional parameters if available
	// Overwrite with new parameters when available
	for _, params := range extraParams {
		// Extract parameters unrelated to upload
		if tokenStr, ok := params["tokens"]; ok {
			token, err := strconv.ParseFloat(fmt.Sprintf("%v", tokenStr), 64)
			require.Nil(t, err)
			faucetTokens = token
			delete(params, "tokens")
		}
		for k, v := range params {
			options[k] = v
		}
	}

	options["lock"] = faucetTokens / 2

	t.Log("Creating new allocation...", options)

	t.Log("Faucet Tokens : ", faucetTokens)

	// First create a wallet and run faucet command
	output, err := CreateWalletForName(t, cliConfigFilename, walletName)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	output, err = ExecuteFaucetWithTokensForWallet(t, walletName, cliConfigFilename, faucetTokens+10)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	output, err = CreateNewAllocationForWallet(t, walletName, cliConfigFilename, CreateParams(options))
	require.Nil(t, err, "create new allocation failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	// Get the allocation ID and return it
	allocationID, err := GetAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}

func DownloadFile(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return downloadFileForWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func downloadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 15*time.Second) // TODO replace with pollers
	t.Logf("Downloading file...")
	cmd := fmt.Sprintf(
		"./zbox download %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func CreateNewAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return CreateNewAllocationForWallet(t, EscapedTestName(t), cliConfigFilename, params)
}

func CreateNewAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Creating new allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"), 3, time.Second*5)
}

func CancelAllocation(t *test.SystemTest, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	t.Logf("Canceling allocation...")
	cmd := fmt.Sprintf(
		"./zbox alloc-cancel --allocation %s "+
			"--wallet %s --configDir ./config --config %s",
		allocationID,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func CreateFileWithSize(name string, size int64) error {
	buffer := make([]byte, size)
	rand.Read(buffer) //nolint:gosec,revive
	return os.WriteFile(name, buffer, os.ModePerm)
}

func GenerateRandomTestFileName(t *test.SystemTest) string {
	path := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))

	//FIXME: Filenames longer than 100 characters are rejected see https://github.com/0chain/zboxcli/issues/249
	randomFilename := cliutils.RandomAlphaNumericString(10)
	return fmt.Sprintf("%s%s%s_test.txt", path, string(os.PathSeparator), randomFilename)
}

func GetAllocationID(str string) (string, error) {
	match := createAllocationRegex.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func UploadFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return uploadFileForWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func uploadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Uploading file...")

	p := CreateParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func GetAllocation(t *test.SystemTest, allocationID string) (allocation climodel.Allocation) {
	output, err := getAllocationWithRetry(t, configPath, allocationID, 1)
	require.Nil(t, err, "error fetching allocation")
	require.Greater(t, len(output), 0, "gettting allocation - output is empty unexpectedly")
	err = json.Unmarshal([]byte(output[0]), &allocation)
	require.Nil(t, err, "error unmarshalling allocation json")
	return
}

func getAllocationWithRetry(t *test.SystemTest, cliConfigFilename, allocationID string, retry int) ([]string, error) {
	t.Logf("Get Allocation...")
	output, err := cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox getallocation --allocation %s --json --silent --wallet %s --configDir ./config --config %s",
		allocationID,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename), retry, time.Second*5)
	return output, err
}

func UpdateAllocation(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return UpdateAllocationWithWallet(t, EscapedTestName(t), cliConfigFilename, params, retry)
}

func UpdateAllocationWithWallet(t *test.SystemTest, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Updating allocation...")
	cmd := fmt.Sprintf(
		"./zbox updateallocation %s --silent --wallet %s "+
			"--configDir ./config --config %s --lock 0.2",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
