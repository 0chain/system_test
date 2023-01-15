package cli_tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func getReadPoolUpdate(t *test.SystemTest, erp climodel.ReadPoolInfo, retry int) (climodel.ReadPoolInfo, error) {
	if retry == 0 {
		retry = 1
	}
	// Wait for read markers to be redeemed
	for i := 0; i < retry; i++ {
		readPool := getReadPoolInfo(t)
		if readPool.Balance == erp.Balance {
			continue
		}

		cliutils.Wait(t, time.Second*30)
		return getReadPoolInfo(t), nil
	}

	return erp, fmt.Errorf("no update found in readpool")
}

func getReadPoolInfo(t *test.SystemTest) climodel.ReadPoolInfo {
	output, err := readPoolInfo(t, configPath)
	require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var readPool climodel.ReadPoolInfo
	err = json.Unmarshal([]byte(output[0]), &readPool)
	require.Nil(t, err, "Error unmarshalling read pool %s", strings.Join(output, "\n"))
	return readPool
}

func getMetaData(t *test.SystemTest, params map[string]interface{}) *climodel.FileMetaResult {
	return getMetaDataWithWallet(t, escapedTestName(t), params)
}

func getMetaDataWithWallet(t *test.SystemTest, walletName string, params map[string]interface{}) *climodel.FileMetaResult {
	output, err := getFileMetaWithWallet(t, walletName, configPath, createParams(params), true)
	require.Nil(t, err, "Error in getting file meta data", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error in getting file meta data - Unexpected number of output lines", strings.Join(output, "\n"))

	var meta climodel.FileMetaResult
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
	require.Nil(t, err, "Failed to unmarshal the json result into FileMetaResult", strings.Join(output, "\n"))
	return &meta
}

func deleteFile(t *test.SystemTest, walletName, params string, retry bool) ([]string, error) {
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
