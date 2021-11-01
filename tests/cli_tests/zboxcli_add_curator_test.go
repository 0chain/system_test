package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestAddCurator(t *testing.T) {
	//t.Parallel()

	t.Run("Add Curator - Should Work", func(t *testing.T) {
		//t.Parallel()

		curatorWalletName := escapedTestName(t) + "_CURATOR"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, curatorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		// clientWallet, err := getWalletForName(t, configPath, escapedTestName(t))
		// require.Nil(t, err, "Error occurred when retrieving client wallet")

		curatorWallet, err := getWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 0, len(allocation.Curators), "Curator list must be empty at the beginning")

		output, err = addCurator(t, allocationID, curatorWallet.ClientID)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("%s added %s as a curator to allocation %s", curatorWallet.ClientID, curatorWallet.ClientID, allocationID)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		allocation = getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, curatorWallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")

	})

}

func addCurator(t *testing.T, allocationId, curatorId string) ([]string, error) {
	t.Logf("Add Curator...")
	params := createParams(map[string]interface{}{"allocation": allocationId, "curator": curatorId})
	cmd := fmt.Sprintf(
		"./zbox addcurator %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		escapedTestName(t)+"_wallet.json",
		configPath,
	)
	return cliutils.RunCommand(cmd)
}
