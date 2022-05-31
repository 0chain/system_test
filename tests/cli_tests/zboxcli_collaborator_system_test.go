package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCollaborator(t *testing.T) {

	t.Run("Add Collaborator _ collaborator client id must be added to file collaborators list", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")
	})

	t.Run("Add Collaborator _ collaborator can NOT be added to a directory", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		dirName := "/NewFolder"
		output, err = createDir(t, configPath, allocationID, dirName, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "/NewFolder directory created", output[0])

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirName,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 0, "Directory collaborators list expected to be empty")

		// Add collaborator to directory
		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": dirName,
		}), false)
		require.NotNil(t, err, "Unexpected success in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected output length", strings.Join(output, "\n"))
		expectedOutput := "add_collaborator_failed: Failed to add collaborator on all blobbers."
		require.Equal(t, expectedOutput, output[0], "Unexpected output in add collaborator", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator must be able to read the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		readPool := getReadPoolInfo(t, allocationID)
		require.Len(t, readPool, 1, "Read pool must exist")
		require.Equal(t, ConvertToValue(0.4), readPool[0].Balance, "Read Pool balance must be equal to locked amount")

		output, err = downloadFileForWallet(t, collaboratorWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)

		require.Nil(t, err, "Error in downloading the file as collaborator", strings.Join(output, "\n"))
		defer os.Remove("tmp" + remotepath)
		require.Len(t, output, 2, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(localpath))
		require.Equal(t, expectedOutput, output[1], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator must not be able to share the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = shareFileWithWallet(t, collaboratorWalletName, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		})
		require.NotNil(t, err, "expected error when collaborator shares a file")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("consensus not reached"), output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Remove Collaborator _ collaborator client id must be removed from file collaborators list", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = removeCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in deleting collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput = fmt.Sprintf("Collaborator %s removed successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta = getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 0, "Collaborator must be removed from file collaborators list")
	})

	t.Run("Remove Collaborator _ file shouldn't be accessible by collaborator anymore", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		// Lock tokens in read pool
		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		output, err = removeCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in deleting collaborator", strings.Join(output, "\n"))

		meta = getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 0, "Collaborator must be removed from file collaborators list")

		output, err = downloadFileForWallet(t, collaboratorWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), false)
		require.NotNil(t, err, "The command must fail since the wallet is not collaborator anymore", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "Error in file operation: No minimum consensus for file meta data of file", output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator to a file owned by somebody else must fail", func(t *testing.T) {
		ownerWalletName := escapedTestName(t) + "_owner"
		anotherWalletName := escapedTestName(t) + "_another"
		t.Skip("piers")
		allocationID := setupAllocationWithWallet(t, ownerWalletName, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		output, err := registerWalletForName(t, configPath, anotherWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, anotherWalletName, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		anotherWallet, err := getWalletForName(t, configPath, anotherWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		localpath := uploadRandomlyGeneratedFileWithWallet(t, ownerWalletName, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaboratorWithWallet(t, anotherWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   anotherWallet.ClientID,
			"remotepath": remotepath,
		}), false)
		require.NotNil(t, err, "Add collaborator must fail since the wallet is not the file owner", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "add_collaborator_failed: Failed to add collaborator on all blobbers.", output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Remove Collaborator from a file owned by somebody else must fail", func(t *testing.T) {
		ownerWalletName := escapedTestName(t) + "_owner"
		anotherWalletName := escapedTestName(t) + "_another"
		t.Skip("piers")
		allocationID := setupAllocationWithWallet(t, ownerWalletName, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		output, err := registerWalletForName(t, configPath, anotherWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokensForWallet(t, anotherWalletName, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		localpath := uploadRandomlyGeneratedFileWithWallet(t, ownerWalletName, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		thirdPersonWalletAddress := "someone_wallet_address"

		output, err = addCollaboratorWithWallet(t, ownerWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   thirdPersonWalletAddress,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", thirdPersonWalletAddress, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta := getMetaDataWithWallet(t, ownerWalletName, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, thirdPersonWalletAddress, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		// Now we test if another wallet can remove from collaborators' list
		output, err = removeCollaboratorWithWallet(t, anotherWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   thirdPersonWalletAddress,
			"remotepath": remotepath,
		}), false)
		require.NotNil(t, err, "Remove collaborator must fail since the wallet is not the file owner", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "remove_collaborator_failed: Failed to remove collaborator on all blobbers.", output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ Collaborator should NOT be able to add another collaborator", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "error in getting wallet", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", 128*KB)
		remotepath := "/" + filepath.Base(localpath)

		thirdPersonWalletAddress := "someone_wallet_address"

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Collaborator %s added successfully for the file %s", collaboratorWallet.ClientID, remotepath)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		// Now we test if collaborator can add another collaborator to filr
		output, err = addCollaboratorWithWallet(t, collaboratorWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   thirdPersonWalletAddress,
			"remotepath": remotepath,
		}), false)
		require.NotNil(t, err, "Add collaborator must fail since the collaborator is not the file owner", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "add_collaborator_failed: Failed to add collaborator on all blobbers.", output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to update the file attributes", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = updateFileAttributesWithWallet(t, configPath, collaboratorWalletName, map[string]interface{}{
			"allocation":         allocationID,
			"remotepath":         remotepath,
			"who-pays-for-reads": "3rd_party",
		}, false)
		require.NotNil(t, err, "Unexpected success in updating the file attributes as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := "updating file attributes: Update attributes failed: request failed, operation failed"
		require.Equal(t, expectedOutput, output[0], "Unexpected output when updating the file attributes", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to rename the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = renameFileWithWallet(t, configPath, collaboratorWalletName, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destname":   "new_name.txt",
		})
		require.NotNil(t, err, "Unexpected success in renaming the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := "Rename failed: Rename request failed. Operation failed."
		require.Equal(t, expectedOutput, output[0], "Unexpected output when renaming the file", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to delete the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = deleteFile(t, collaboratorWalletName, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
		}), false)
		require.NotNil(t, err, "Unexpected success in deleting the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Delete failed. Delete failed: Success_rate", "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to move the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = moveFileWithWallet(t, collaboratorWalletName, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destpath":   "/",
		}, true)
		require.Nil(t, err, "Error in moving the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := "Copy failed: Copy request failed. Operation failed."
		require.Equal(t, expectedOutput, output[0], "Unexpected output when removing the file", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should be able to update the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		updatedLocalPath := generateRandomTestFileName(t)
		err = createFileWithSize(updatedLocalPath, 128*KB)
		require.Nil(t, err)

		output, err = updateFileWithWallet(t, collaboratorWalletName, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  updatedLocalPath,
		}, true)

		defer os.Remove(updatedLocalPath)
		require.Nil(t, err, "failed in updating the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 2, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(localpath))
		require.Equal(t, expectedOutput, output[1], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to copy the file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/tmp", 128*KB)
		remotepath := "/tmp/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = copyFileForWallet(t, configPath, collaboratorWalletName, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destpath":   "/",
		}, true)
		require.Nil(t, err, "Unexpected success in copying the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := "Copy failed: Copy request failed. Operation failed."
		require.Equal(t, expectedOutput, output[0], "Unexpected output", strings.Join(output, "\n"))
	})

	t.Run("Add Collaborator _ collaborator should NOT be able to download encrypted file", func(t *testing.T) {
		collaboratorWalletName := escapedTestName(t) + "_collaborator"
		t.Skip("piers")
		output, err := registerWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		collaboratorWallet, err := getWalletForName(t, configPath, collaboratorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 2 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		localpath := generateRandomTestFileName(t)
		err = createFileWithSize(localpath, 128*KB)
		require.Nil(t, err)
		defer os.Remove(localpath)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filepath.Base(localpath),
			"localpath":  localpath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile(`Status completed callback. Type = application/octet-stream. Name = (?P<Filename>.+)`), output[1])

		remotepath := "/" + filepath.Base(localpath)

		output, err = addCollaborator(t, createParams(map[string]interface{}{
			"allocation": allocationID,
			"collabid":   collaboratorWallet.ClientID,
			"remotepath": remotepath,
		}), true)
		require.Nil(t, err, "error in adding collaborator", strings.Join(output, "\n"))

		meta := getMetaData(t, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"json":       "",
		})
		require.Len(t, meta.Collaborators, 1, "Collaborator must be added in file collaborators list")
		require.Equal(t, collaboratorWallet.ClientID, meta.Collaborators[0].ClientID, "Collaborator must be added in file collaborators list")

		output, err = readPoolLock(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "1h",
		}), true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
		require.Len(t, output, 1, "Unexpected number of output lines", strings.Join(output, "\n"))
		require.Equal(t, "locked", output[0])

		readPool := getReadPoolInfo(t, allocationID)
		require.Len(t, readPool, 1, "Read pool must exist")
		require.Equal(t, ConvertToValue(0.4), readPool[0].Balance, "Read Pool balance must be equal to locked amount")

		output, err = downloadFileForWallet(t, collaboratorWalletName, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  "tmp/",
		}), true)
		require.NotNil(t, err, "Unexpected success in downloading the file as collaborator", strings.Join(output, "\n"))
		require.Len(t, output, 2, "Unexpected number of output lines", strings.Join(output, "\n"))
		expectedOutput := "Error in file operation: File content didn't match with uploaded file"
		require.Equal(t, expectedOutput, output[1], "Unexpected output", strings.Join(output, "\n"))
	})
}

func getReadPoolInfo(t *testing.T, allocationID string) []climodel.ReadPoolInfo {
	output, err := readPoolInfo(t, configPath, allocationID)
	require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	readPool := []climodel.ReadPoolInfo{}
	err = json.Unmarshal([]byte(output[0]), &readPool)
	require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))
	return readPool
}

func getMetaData(t *testing.T, params map[string]interface{}) *climodel.FileMetaResult {
	return getMetaDataWithWallet(t, escapedTestName(t), params)
}

func getMetaDataWithWallet(t *testing.T, walletName string, params map[string]interface{}) *climodel.FileMetaResult {
	output, err := getFileMetaWithWallet(t, walletName, configPath, createParams(params), true)
	require.Nil(t, err, "Error in getting file meta data", strings.Join(output, "\n"))
	require.Len(t, output, 1, "Error in getting file meta data - Unexpected number of output lines", strings.Join(output, "\n"))

	var meta climodel.FileMetaResult
	err = json.NewDecoder(strings.NewReader(output[0])).Decode(&meta)
	require.Nil(t, err, "Failed to unmarshal the json result into FileMetaResult", strings.Join(output, "\n"))
	return &meta
}

func addCollaborator(t *testing.T, params string, retry bool) ([]string, error) {
	return addCollaboratorWithWallet(t, escapedTestName(t), params, retry)
}

func addCollaboratorWithWallet(t *testing.T, walletName, params string, retry bool) ([]string, error) {
	t.Logf("Adding collaborator...")
	cmd := fmt.Sprintf(
		"./zbox add-collab %s --silent --wallet %s "+
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

func removeCollaborator(t *testing.T, params string, retry bool) ([]string, error) {
	return removeCollaboratorWithWallet(t, escapedTestName(t), params, retry)
}

func removeCollaboratorWithWallet(t *testing.T, walletName, params string, retry bool) ([]string, error) {
	t.Logf("Removing collaborator...")
	cmd := fmt.Sprintf(
		"./zbox delete-collab %s --silent --wallet %s "+
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

func deleteFile(t *testing.T, walletName, params string, retry bool) ([]string, error) {
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
