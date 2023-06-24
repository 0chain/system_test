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
	"path/filepath"
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
	output, err := readPoolLock(t, cliConfigFilename, readPoolParams, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	return allocationID
}

func readPoolLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
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

func generateChecksum(t *test.SystemTest, filePath string) string {
	t.Logf("Generating checksum for file [%v]...", filePath)

	output, err := cliutils.RunCommandWithoutRetry("shasum -a 256 " + filePath)
	require.Nil(t, err, "Checksum generation for file %v failed", filePath, strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)

	matcher := regexp.MustCompile("(.*) " + filePath + "$")
	require.Regexp(t, matcher, output[0], "Checksum execution output did not match expected", strings.Join(output, "\n"))

	return matcher.FindAllStringSubmatch(output[0], 1)[0][1]
}

func setupWallet(t *test.SystemTest, configPath string) []string {
	output, err := CreateWallet(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	output, err = ExecuteFaucetWithTokens(t, configPath, 1)
	require.Nil(t, err, strings.Join(output, "\n"))

	output, err = getBalance(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
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

func createNewAllocationWithoutRetry(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		EscapedTestName(t)+"_allocation.txt"))
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

func extractAuthToken(str string) (string, error) {
	match := reAuthToken.FindStringSubmatch(str)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", errors.New("auth token did not match")
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

func shareFolderInAllocation(t *test.SystemTest, cliConfigFilename, param string) ([]string, error) {
	return shareFolderInAllocationForWallet(t, EscapedTestName(t), cliConfigFilename, param)
}

func shareFolderInAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string) ([]string, error) {
	t.Logf("Sharing file/folder...")
	cmd := fmt.Sprintf(
		"./zbox share %s --silent --wallet %s --configDir ./config --config %s",
		param,
		wallet+"_wallet.json",
		cliConfigFilename,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func listFilesInAllocation(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return listFilesInAllocationForWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func listFilesInAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 10*time.Second) // TODO replace with poller
	t.Logf("Listing individual file in allocation...")
	cmd := fmt.Sprintf(
		"./zbox list %s --silent --wallet %s --configDir ./config --config %s",
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

func listAllFilesInAllocation(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	cliutils.Wait(t, 10*time.Second) // TODO replace with poller
	t.Logf("Listing all files in allocation...")
	cmd := fmt.Sprintf(
		"./zbox list-all %s --silent --wallet %s --configDir ./config --config %s",
		param,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func GetAllocationID(str string) (string, error) {
	match := createAllocationRegex.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func uploadWithParam(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}) {
	uploadWithParamForWallet(t, EscapedTestName(t), cliConfigFilename, param)
}
func uploadWithParamForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}) {
	filename, ok := param["localpath"].(string)
	require.True(t, ok)

	output, err := uploadFileForWallet(t, wallet, cliConfigFilename, param, true)
	require.Nil(t, err, "Upload file failed due to error ", err, strings.Join(output, "\n"))

	require.Len(t, output, 2)

	aggregatedOutput := strings.Join(output, " ")
	require.Contains(t, aggregatedOutput, StatusCompletedCB)
	require.Contains(t, aggregatedOutput, filepath.Base(filename))
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

func uploadFileWithoutRetry(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Uploading file...")
	p := CreateParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		p,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func generateFileAndUpload(t *test.SystemTest, allocationID, remotepath string, size int64) string {
	return generateFileAndUploadForWallet(t, EscapedTestName(t), allocationID, remotepath, size)
}

func generateFileAndUploadForWallet(t *test.SystemTest, wallet, allocationID, remotepath string, size int64) string {
	filename := GenerateRandomTestFileName(t)

	err := CreateFileWithSize(filename, size)
	require.Nil(t, err)

	// Upload parameters
	uploadWithParamForWallet(t, wallet, configPath, map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	})

	return filename
}

func generateFileAndUploadWithParam(t *test.SystemTest, allocationID, remotepath string, size int64, params map[string]interface{}) string {
	filename := GenerateRandomTestFileName(t)

	err := CreateFileWithSize(filename, size)
	require.Nil(t, err)

	p := map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	}

	for k, v := range params {
		p[k] = v
	}

	// Upload parameters
	uploadWithParam(t, configPath, p)

	return filename
}

func uploadRandomlyGeneratedFile(t *test.SystemTest, allocationID, remotePath string, fileSize int64) string {
	return uploadRandomlyGeneratedFileWithWallet(t, EscapedTestName(t), allocationID, remotePath, fileSize)
}

func uploadRandomlyGeneratedFileWithWallet(t *test.SystemTest, walletName, allocationID, remotePath string, fileSize int64) string {
	filename := GenerateRandomTestFileName(t)
	err := CreateFileWithSize(filename, fileSize)
	require.Nil(t, err)

	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}

	output, err := uploadFileForWallet(t, walletName, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotePath + filepath.Base(filename),
		"localpath":  filename,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 2)
	require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])
	return filename
}

func moveAllocationFile(t *test.SystemTest, allocationID, remotepath, destination string) { // nolint
	output, err := moveFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destpath":   "/" + destination,
	}, true)
	require.Nil(t, err, "error in moving the file: ", strings.Join(output, "\n"))
}

func renameAllocationFile(t *test.SystemTest, allocationID, remotepath, newName string) {
	output, err := renameFile(t, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": "/" + remotepath,
		"destname":   newName,
	}, true)
	require.Nil(t, err, "error in renaming the file: ", strings.Join(output, "\n"))
}

func updateFileWithRandomlyGeneratedData(t *test.SystemTest, allocationID, remotepath string, size int64) string {
	return updateFileWithRandomlyGeneratedDataWithWallet(t, EscapedTestName(t), allocationID, remotepath, size)
}

func updateFileWithRandomlyGeneratedDataWithWallet(t *test.SystemTest, walletName, allocationID, remotepath string, size int64) string {
	localfile := GenerateRandomTestFileName(t)
	err := CreateFileWithSize(localfile, size)
	require.Nil(t, err)

	output, err := updateFileWithWallet(t, walletName, configPath, map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remotepath,
		"localpath":  localfile,
	}, true)
	require.Nil(t, err, strings.Join(output, "\n"))
	return localfile
}

func renameFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Renaming file...")
	p := CreateParams(param)
	cmd := fmt.Sprintf(
		"./zbox rename %s --silent --wallet %s --configDir ./config --config %s",
		p,
		EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func updateFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return updateFileWithWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func updateFileWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Updating file...")

	p := CreateParams(param)
	cmd := fmt.Sprintf(
		"./zbox update %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
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

// ConvertToToken converts the value to ZCN tokens
func ConvertToToken(value int64) float64 {
	return float64(value) / float64(TOKEN_UNIT)
}

// ConvertToValue converts ZCN tokens to value
func ConvertToValue(token float64) int64 {
	return int64(token * float64(TOKEN_UNIT))
}

func moveFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return moveFileWithWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func moveFileWithWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Moving file...")
	p := CreateParams(param)
	cmd := fmt.Sprintf(
		"./zbox move %s --silent --wallet %s --configDir ./config --config %s",
		p,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func setupWalletWithCustomTokens(t *test.SystemTest, configPath string, tokens float64) []string {
	output, err := CreateWallet(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	ExecuteFaucetWithTokens(t, configPath, tokens)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
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
