package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCollaborator(t *testing.T) {
	t.Parallel()

	t.Run("Add Collaborator _ collaborator must be able to read the file", func(t *testing.T) {
		t.Parallel()

		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}))
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getFileMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Equal(t, 1, len(meta.Collaborators), "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		}))
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		output, err = downloadFileForWallet(t, collaboratorWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}))
		require.Nil(t, err, "Error in downloading the file as collaborator", strings.Join(output, "\n"))
		defer os.Remove("tmp" + remotepath)
		require.Equal(t, 2, len(output), "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(localpath))
		require.Equal(t, expectedOutput, output[1], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator client id must be added to file collaborators list", func(t *testing.T) {
		t.Parallel()

		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}))
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta := getFileMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Equal(t, 1, len(meta.Collaborators), "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")
	})
}

func getReadPoolInfo(t *testing.T, allocationID string) []climodel.ReadPoolInfo {
	output, err := readPoolInfo(t, configPath, allocationID)
	require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

	readPool := []climodel.ReadPoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &readPool)
	require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
	return readPool
}

func getFileMetaData(t *testing.T, params map[string]interface{}) *climodel.FileMetaResult {
	output, err := getFileMeta(t, configPath, createParams(params))
	require.Nil(t, err, "Error in getting file meta data", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error in getting file meta data - Unexpected number of output lines", strings.Join(output, "\n"))

	var meta climodel.FileMetaResult
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
	require.Nil(t, err, "Failed to unmarshal the json result into FileMetaResult", strings.Join(output, "\n"))
	return &meta
}

func addCollaborator(t *testing.T, params string) ([]string, error) {
	return addCollaboratorWithWallet(t, escapedTestName(t), params)
}

func addCollaboratorWithWallet(t *testing.T, walletName, params string) ([]string, error) {
	t.Logf("Adding collaborator...")
	cmd := fmt.Sprintf(
		"./zbox add-collab %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		walletName+"_wallet.json",
		configPath,
	)
	return cliutils.RunCommand(cmd)
}
