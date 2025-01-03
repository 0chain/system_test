package tokenomics_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/tests/tokenomics_tests/utils"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateEnterpriseAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Run("Create enterprise allocation with blobber auth tickets should pass", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		// Create enterprise allocation with blobber auth tickets
		options := map[string]interface{}{
			"size":                 "1024",
			"lock":                 "0.5",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = utils.CreateNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation for locking cost equal to the cost calculated should work", func(t *test.SystemTest) {
		_, _ = utils.CreateWallet(t, configPath)

		blobberAuthTickets, blobbersIds := utils.GenerateBlobberAuthTickets(t, configPath)
		options := map[string]interface{}{
			"cost":                 "",
			"size":                 "10000",
			"read_price":           "0-1",
			"write_price":          "0-1",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobbersIds,
		}
		output, err := createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		options = map[string]interface{}{
			"lock":                 allocationCost,
			"size":                 "10000",
			"read_price":           "0-1",
			"write_price":          "0-1",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobbersIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with smallest possible size (1024) Should Work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"size":                 "1024",
			"lock":                 "0.5",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation for another owner should Work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		output, err = utils.CreateWalletForName(t, configPath, utils.EscapedTestName(t)+"_other")
		require.Nil(t, err, "Error creating other wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing facuet")

		targetWalletName := utils.EscapedTestName(t)
		targetWallet, err := utils.GetWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "could not get target wallet")

		_, err = utils.ExecuteFaucetWithTokensForWallet(t, utils.EscapedTestName(t)+"_other", configPath, 1000)
		require.Nil(t, err, "Error executing facuet")

		_, err = utils.GetWalletForName(t, configPath, utils.EscapedTestName(t)+"_other")
		require.Nil(t, err, "could not get other wallet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"lock":                 "0.5",
			"owner":                targetWallet.ClientID,
			"owner_public_key":     targetWallet.ClientPublicKey,
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		file := utils.GenerateRandomTestFileName(t)
		fileSize := int64(102400)
		err = utils.CreateFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": "/",
			"encrypt":    "",
		}
		output, err = utils.UploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = utils.UploadFileForWallet(t, utils.EscapedTestName(t)+"_other", configPath, uploadParams, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, ""), "Operation needs to be performed by the owner or the payer of the allocation")

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with parity specified Should Work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucets")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"size":                 "1024",
			"parity":               "1",
			"lock":                 "0.5",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with data specified Should Work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"size":                 "1024",
			"data":                 "1",
			"lock":                 "0.5",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with write price range Should Work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"size":                 "1024",
			"write_price":          "0-9999",
			"lock":                 "0.5",
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}

		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation should have all file options permitted by default", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err)

		var alloc climodel.Allocation

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(63), alloc.FileOptions)
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation with some forbidden file options flags should pass and show in allocation", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_upload": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc := utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(62), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_delete": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(61), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_update": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(59), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_move": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(55), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_copy": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(47), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "forbid_rename": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)
		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, uint16(31), alloc.FileOptions)

		createEnterpriseAllocationTestTeardown(t, allocationID)
	})

	t.Run("Create enterprise allocation third_party_extendable should be false by default and change with flags", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"lock": "0.5", "size": 1024, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err := utils.GetAllocationID(output[0])
		require.Nil(t, err)

		var alloc climodel.Allocation

		alloc = utils.GetAllocation(t, allocationID)
		require.Equal(t, false, alloc.ThirdPartyExtendable)
		createEnterpriseAllocationTestTeardown(t, allocationID)

		blobberAuthTickets, blobberIds = utils.GenerateBlobberAuthTickets(t, configPath)

		options = map[string]interface{}{"lock": "0.5", "size": 1024, "third_party_extendable": nil, "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "Allocation created", strings.Join(output, "\n"))

		allocationID, err = utils.GetAllocationID(output[0])
		require.Nil(t, err)

		alloc = utils.GetAllocation(t, allocationID)
		require.NotNil(t, alloc, "error fetching allocation")
		require.Equal(t, true, alloc.ThirdPartyExtendable)
		createEnterpriseAllocationTestTeardown(t, allocationID)
	})
	t.Run("Create enterprise allocation for locking cost less than minimum cost should not work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"cost":                 "",
			"read_price":           "0-1",
			"write_price":          "0-1",
			"size":                 10000,
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")

		allocationCost, err := getAllocationCost(output[0])
		require.Nil(t, err, "could not get allocation cost", strings.Join(output, "\n"))

		mustFailCost := allocationCost * 0.8
		options = map[string]interface{}{
			"lock":                 mustFailCost,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
			"enterprise":           true,
		}

		output, err = createNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "not enough tokens to honor the allocation", strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation for locking negative cost should not work", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"cost":                 "",
			"read_price":           "0-1",
			"write_price":          "0-1",
			"size":                 10000,
			"enterprise":           true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds,
		}
		mustFailCost := -1
		options = map[string]interface{}{"lock": mustFailCost}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation without blobber auth tickets should fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet", strings.Join(output, "\n"))

		_, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{
			"size":               "1024",
			"lock":               "0.5",
			"enterprise":         true,
			"preferred_blobbers": blobberIds,
		}
		output, err = utils.CreateNewEnterpriseAllocation(t, configPath, utils.CreateParams(options))

		require.NotNil(t, err, "expected an error when creating allocation without blobber auth tickets %v", err)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Not enough blobbers to honor the allocation", "expected error message to mention missing auth tickets %v", strings.Join(output, "\n"))
	})
	t.Run("Create enterprise allocation with too large parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"parity": "99", "lock": "0.5", "size": 1024, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets, "preferred_blobbers": blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "blobbers provided are not enough to honour the allocation") //nolint
	})

	t.Run("Create enterprise allocation with too large data (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"data": "99", "lock": "0.5", "size": 1024, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers": blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "blobbers provided are not enough to honour the allocation") //nolint
	})

	t.Run("Create enterprise allocation with too large data and parity (Greater than the number of blobbers) Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"data": "30", "parity": "20", "lock": "0.5", "size": 1024, "enterprise": true, "blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers": blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Contains(t, output[0], "blobbers provided are not enough to honour the allocation") //nolint
	})

	t.Run("Create enterprise allocation with size smaller than limit (size < 1024) Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"size": 256, "lock": "0.5", "enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Equal(t, "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation size", output[0], strings.Join(output, "\n"))
	})

	t.Run("Create enterprise allocation with no parameter (missing lock) Should Fail", func(t *test.SystemTest) {
		output, err := utils.CreateWallet(t, configPath)
		require.Nil(t, err, "Error registering wallet", strings.Join(output, "\n"))

		_, err = utils.ExecuteFaucetWithTokens(t, configPath, 10)
		require.Nil(t, err, "Error executing faucet")

		blobberAuthTickets, blobberIds := utils.GenerateBlobberAuthTickets(t, configPath)

		options := map[string]interface{}{"enterprise": true,
			"blobber_auth_tickets": blobberAuthTickets,
			"preferred_blobbers":   blobberIds}
		output, err = createNewEnterpriseAllocationWithoutRetry(t, configPath, utils.CreateParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Equal(t, "missing required 'lock' argument", output[len(output)-1])
	})
}

func createNewEnterpriseAllocation(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return createNewEnterpriseAllocationForWallet(t, utils.EscapedTestName(t), cliConfigFilename, params)
}

func createNewEnterpriseAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Creating new allocation...")
	return cliutils.RunCommand(t, fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		wallet+"_allocation.txt"), 3, time.Second*5)
}

func createNewEnterpriseAllocationWithoutRetry(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./zbox newallocation %s --silent --wallet %s --configDir ./config --config %s --allocationFileName %s",
		params,
		utils.EscapedTestName(t)+"_wallet.json",
		cliConfigFilename,
		utils.EscapedTestName(t)+"_allocation.txt"))
}

func createEnterpriseAllocationTestTeardown(t *test.SystemTest, allocationID string) {
	t.Cleanup(func() {
		_, _ = cancelAllocation(t, configPath, allocationID, false)
	})
}
