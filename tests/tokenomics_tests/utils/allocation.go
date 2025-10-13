package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	coreClient "github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/zcncore"

	"github.com/0chain/gosdk/core/conf"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

var (
	createAllocationRegex = regexp.MustCompile(`^Allocation created: (.+)$`)
)

func SetupEnterpriseAllocationAndReadLock(t *test.SystemTest, cliConfigFilename string, extraParam map[string]interface{}) string {
	allocationID := SetupEnterpriseAllocation(t, cliConfigFilename, extraParam)
	return allocationID
}

func SetupAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return SetupAllocationWithWallet(t, EscapedTestName(t), cliConfigFilename, extraParams...)
}

func SetupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	faucetTokens := 2.0
	lockAmountPassed := false
	// Then create new allocation
	options := map[string]interface{}{"size": "10000", "lock": "0.5"}

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

		if _, lockPassed := params["lock"]; lockPassed {
			lockAmountPassed = true
		}

		for k, v := range params {
			options[k] = v
		}
	}

	if !lockAmountPassed {
		options["lock"] = faucetTokens / 2
	}

	t.Log("Creating new allocation...", options)

	t.Log("Faucet Tokens : ", faucetTokens)

	// First create a wallet and run faucet command
	output, err := CreateWalletForName(t, cliConfigFilename, walletName)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))

	output, err = ExecuteFaucetWithTokensForWallet(t, walletName, cliConfigFilename, faucetTokens+10)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	output, err = CreateNewAllocationForWallet(t, walletName, cliConfigFilename, CreateParams(options))
	require.Nil(t, err, "create new allocation failed", strings.Join(output, "\n"))

	// Get the allocation ID and return it
	allocationID, err := GetAllocationID(output[len(output)-1])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}

func SetupEnterpriseAllocation(t *test.SystemTest, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	return SetupEnterpriseAllocationWithWallet(t, EscapedTestName(t), cliConfigFilename, extraParams...)
}

func SetupEnterpriseAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) string {
	if len(extraParams) == 0 {
		extraParams = append(extraParams, map[string]interface{}{})
	}

	extraParams[0]["blobber_auth_tickets"], extraParams[0]["preferred_blobbers"] = GenerateBlobberAuthTicketsWithWallet(t, walletName, cliConfigFilename)
	extraParams[0]["enterprise"] = true

	allocID := SetupAllocationWithWallet(t, walletName, cliConfigFilename, extraParams...)

	t.Logf("Enterprise allocation created with ID: %s", allocID)

	return allocID
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
func CreateNewEnterpriseAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return CreateNewEnterpriseAllocationForWallet(t, EscapedTestName(t), cliConfigFilename, params)
}

func CreateNewEnterpriseAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Creating new enterprise allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s --enterprise",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"), 3, time.Second*5)
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
	_, err := rand.Read(buffer)
	if err != nil {
		return err
	} //nolint:gosec,revive
	return os.WriteFile(name, buffer, os.ModePerm) //nolint:gosec
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
	return UploadFileForWallet(t, EscapedTestName(t), cliConfigFilename, param, retry)
}

func UploadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
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

func DeleteFile(t *test.SystemTest, walletName, params string, retry bool) ([]string, error) {
	t.Logf("Deleting file...")
	cmd := fmt.Sprintf(
		"./zbox delete %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		walletName+"_wallet.json",
		configPath,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getBlobberNotPartOfAllocation(walletname, configFile, allocationID string) (*sdk.Blobber, error) {
	err := InitSDK(walletname, configFile)
	if err != nil {
		return nil, err
	}

	a, err := sdk.GetAllocation(allocationID)
	if err != nil {
		return nil, err
	}

	blobbers, err := sdk.GetBlobbers(true, false)
	if err != nil {
		return nil, err
	}

	allocationBlobsMap := map[string]bool{}
	for _, b := range a.BlobberDetails {
		allocationBlobsMap[b.BlobberID] = true
	}

	for _, blobber := range blobbers {
		if _, ok := allocationBlobsMap[string(blobber.ID)]; !ok {
			return blobber, nil
		}
	}

	return nil, fmt.Errorf("failed to get blobber not part of allocation")
}

// GetBlobberNotPartOfAllocation returns a blobber not part of current allocation
func GetBlobberIdAndUrlNotPartOfAllocation(walletname, configFile, allocationID string) (blobberId, blobberUrl string, err error) {
	blobber, err := getBlobberNotPartOfAllocation(walletname, configFile, allocationID)
	if err != nil || blobber == nil {
		return "", "", err
	}
	return string(blobber.ID), blobber.BaseURL, err
}

func InitSDK(wallet, configFile string) error {
	f, err := os.Open(wallet)
	if err != nil {
		return err
	}
	clientBytes, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	walletJSON := string(clientBytes)

	parsed, err := conf.LoadConfigFile(configFile)
	if err != nil {
		return err
	}

	err = coreClient.Init(context.Background(), parsed)
	if err != nil {
		return err
	}

	err = coreClient.InitSDK(
		"{}",
		parsed.BlockWorker,
		parsed.ChainID,
		parsed.SignatureScheme,
		0, true,
	)
	if err != nil {
		return err
	}

	err = zcncore.SetGeneralWalletInfo(walletJSON, parsed.SignatureScheme)
	if err != nil {
		log.Println("Error in sdk init", err)
		return err
	}

	if coreClient.GetClient().IsSplit {
		zcncore.RegisterZauthServer(parsed.ZauthServer)
	}

	return err
}
