package cli_tests

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCollaborator(t *testing.T) {
	t.Parallel()

	t.Run("Add Collaborator _ Should work", func(t *testing.T) {
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
	})
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
