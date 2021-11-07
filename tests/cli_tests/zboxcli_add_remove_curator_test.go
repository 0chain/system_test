package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestAddCurator(t *testing.T) {
	t.Parallel()

	t.Run("Add Curator _ must fail when the allocation doesn't exist ", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving wallet")

		params := createParams(map[string]interface{}{"allocation": "INVALID ALLOCATION ID", "curator": wallet.ClientID})
		output, err = addCurator(t, params)
		require.NotNil(t, err, "unexpected success on adding curator", strings.Join(output, "\n"))
		require.Contains(t, output[0], "adding curator:[txn] too less sharders to confirm it", strings.Join(output, "\n"))
	})

	t.Run("Add Curator _ attempt to add curator by anyone except allocation owner must fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		anotherClientWalletName := escapedTestName(t) + "_ANOTHER_WALLET"
		output, err = registerWalletForName(configPath, anotherClientWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		anotherWallet, err := getWalletForName(t, configPath, anotherClientWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": anotherWallet.ClientID})
		output, err = addCuratorWithWallet(t, anotherClientWalletName, params)
		require.NotNil(t, err, "unexpected success on adding curator", strings.Join(output, "\n"))
		require.Contains(t, output[0], "adding curator:[txn] too less sharders to confirm it", strings.Join(output, "\n"))
	})

	t.Run("Add Curator _ must fail when 'curator' parameter is missing", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID})
		output, err = addCurator(t, params)
		require.NotNil(t, err, "unexpected success on adding curator", strings.Join(output, "\n"))
		require.Equal(t, "Error: curator flag is missing", output[0], strings.Join(output, "\n"))
	})

	t.Run("Add Curator _ Curator must be able to transfer the allocation ownership", func(t *testing.T) {
		t.Parallel()

		curatorWalletName := escapedTestName(t) + "_CURATOR"
		targetWalletName := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, curatorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		curatorWallet, err := getWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = registerWalletForName(configPath, targetWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": curatorWallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, curatorWallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")

		output, err = transferAllocationOwnershipWithWallet(t, allocationID, curatorWalletName, targetWallet.ClientID, targetWallet.ClientPublicKey)
		require.Nil(t, err, "error in transferring allocation as curator", strings.Join(output, "\n"))
		require.Equal(t, 1, len(output), "unexpected output length:", strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("transferred ownership of allocation %s to %s", allocationID, targetWallet.ClientID)
		require.Equal(t, expectedOutput, output[0], "unexpected output:", strings.Join(output, "\n"))
	})

	t.Run("Add Curator _ Owner added as curator must be able to transfer the ownership", func(t *testing.T) {
		t.Parallel()
		targetWalletName := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, targetWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": wallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, wallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")

		output, err = transferAllocationOwnership(t, allocationID, targetWallet.ClientID, targetWallet.ClientPublicKey)
		require.Nil(t, err, "error in transferring allocation as curator", strings.Join(output, "\n"))
	})

	t.Run("Add Curator _ Owner must be able to add itself as curator", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": wallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, wallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")
	})

	t.Run("Add Curator _ Curator must be added to allocation curators' list", func(t *testing.T) {
		t.Parallel()

		curatorWalletName := escapedTestName(t) + "_CURATOR"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, curatorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

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

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": curatorWallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		expectedOutput := fmt.Sprintf("%s added %s as a curator to allocation %s", curatorWallet.ClientID, curatorWallet.ClientID, allocationID)
		require.Equal(t, expectedOutput, output[0], strings.Join(output, "\n"))

		allocation = getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, curatorWallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")
	})

	t.Run("Remove Curator _ Curator must no longer be able to transfer the allocation ownership", func(t *testing.T) {
		t.Parallel()

		curatorWalletName := escapedTestName(t) + "_CURATOR"
		targetWalletName := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, curatorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		curatorWallet, err := getWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = registerWalletForName(configPath, targetWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": curatorWallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, curatorWallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")

		output, err = removeCurator(t, params)
		require.Nil(t, err, "error in removing curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation = getAllocation(t, allocationID)
		require.Equal(t, 0, len(allocation.Curators), "Curators list must be empty after removing curator")

		output, err = transferAllocationOwnershipWithWallet(t, allocationID, curatorWalletName, targetWallet.ClientID, targetWallet.ClientPublicKey)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error adding curator:[txn] too less sharders to confirm it", strings.Join(output, "\n"))
	})

	t.Run("Remove Curator _ Curator must be removed from the allocation curators list", func(t *testing.T) {
		t.Parallel()

		curatorWalletName := escapedTestName(t) + "_CURATOR"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(configPath, curatorWalletName)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		curatorWallet, err := getWalletForName(t, configPath, curatorWalletName)
		require.Nil(t, err, "Error occurred when retrieving curator wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{"lock": "0.5", "size": 1 * MB}))
		require.Nil(t, err, "create new allocation failed", err, strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", err, strings.Join(output, "\n"))
		defer createAllocationTestTeardown(t, allocationID)

		params := createParams(map[string]interface{}{"allocation": allocationID, "curator": curatorWallet.ClientID})
		output, err = addCurator(t, params)
		require.Nil(t, err, "error in adding curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation := getAllocation(t, allocationID)
		require.Equal(t, 1, len(allocation.Curators), "Curator must've added to the allocation curators list")
		require.Equal(t, curatorWallet.ClientID, allocation.Curators[0], "Curator must've added to the allocation curators list")

		output, err = removeCurator(t, params)
		require.Nil(t, err, "error in removing curator", strings.Join(output, "\n"))

		wait(t, 5*time.Second)

		allocation = getAllocation(t, allocationID)
		require.Equal(t, 0, len(allocation.Curators), "Curators list must be empty after removing curator")
	})
}

func transferAllocationOwnership(t *testing.T, allocationID, newOwner, newOwnerPublicKey string) ([]string, error) {
	return transferAllocationOwnershipWithWallet(t, allocationID, escapedTestName(t), newOwner, newOwnerPublicKey)
}

func transferAllocationOwnershipWithWallet(t *testing.T, allocationID, curatorWalletName, newOwner, newOwnerPublicKey string) ([]string, error) {
	t.Logf("Transaferring allocation ownership...")
	params := createParams(map[string]interface{}{"allocation": allocationID, "new_owner": newOwner, "new_owner_key": newOwnerPublicKey})
	cmd := fmt.Sprintf(
		"./zbox transferallocation %s --silent --wallet %s_wallet.json "+
			"--configDir ./config --config %s",
		params,
		curatorWalletName,
		configPath,
	)
	return cliutils.RunCommand(cmd)
}

func addCurator(t *testing.T, params string) ([]string, error) {
	return addCuratorWithWallet(t, escapedTestName(t), params)
}

func addCuratorWithWallet(t *testing.T, walletName, params string) ([]string, error) {
	t.Logf("Adding curator...")
	cmd := fmt.Sprintf(
		"./zbox addcurator %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		walletName+"_wallet.json",
		configPath,
	)
	return cliutils.RunCommand(cmd)
}

func removeCurator(t *testing.T, params string) ([]string, error) {
	return removeCuratorWithWallet(t, escapedTestName(t), params)
}

func removeCuratorWithWallet(t *testing.T, walletName, params string) ([]string, error) {
	t.Logf("Removing curator...")
	cmd := fmt.Sprintf(
		"./zbox removecurator %s --silent --wallet %s "+
			"--configDir ./config --config %s",
		params,
		walletName+"_wallet.json",
		configPath,
	)
	return cliutils.RunCommand(cmd)
}
