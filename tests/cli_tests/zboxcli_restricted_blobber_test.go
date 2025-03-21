package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk_common/core/zcncrypto"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"
)

func TestRestrictedBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create allocation for locking cost equal to the cost calculated should work")

	t.TestSetup("register wallet and get blobbers", func() {
		createWallet(t)

		// get the list of blobbers
		blobbersList = getBlobbersList(t)
		require.Greater(t, len(blobbersList), 0, "No blobbers found")

		for _, blobber := range blobbersList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"not_available": false,
				"is_restricted": false,
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}
	})

	t.Cleanup(func() {
		for _, blobber := range blobbersList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"not_available": false,
				"is_restricted": false,
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		}
	})

	t.RunSequentiallyWithTimeout("Create allocation on restricted blobbers should pass with correct auth tickets", 10*time.Minute, func(t *test.SystemTest) {
		// Update blobber config to make restricted blobbers to true
		blobber1 := blobbersList[0]
		blobber2 := blobbersList[1]
		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber1.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber2.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber1.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber2.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5", "force": "true", "auth_round_expiry": 1000000000}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

		// Retry with auth ticket
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "could not get wallet")

		blobber1AuthTicket, err := getBlobberAuthTicket(t, blobber1.Id, blobber1.Url, wallet.ClientID)
		require.Nil(t, err, "could not get blobber1 auth ticket")
		blobber2AuthTicket, err := getBlobberAuthTicket(t, blobber2.Id, blobber2.Url, wallet.ClientID)
		require.Nil(t, err, "could not get blobber2 auth ticket")

		var preferredBlobbers, blobberAuthTickets string
		for i, bb := range blobbersList {
			if i == len(blobbersList)-1 {
				preferredBlobbers += bb.Id
				break
			}

			preferredBlobbers += bb.Id + ","
			if i == 0 {
				blobberAuthTickets += blobber1AuthTicket + ","
			} else if i == 1 {
				blobberAuthTickets += blobber2AuthTicket + ","
			} else {
				blobberAuthTickets += ","
			}
		}

		options = map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5", "preferred_blobbers": preferredBlobbers, "blobber_auth_tickets": blobberAuthTickets, "force": "true", "auth_round_expiry": 1000000000}
		output, err = createNewAllocation(t, configPath, createParams(options))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.True(t, len(output) > 0, "expected output length be at least 1")
		require.Regexp(t, regexp.MustCompile("^Allocation created: [0-9a-fA-F]{64}$"), output[0], strings.Join(output, "\n"))

		allocationID, err := getAllocationID(output[0])
		require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))
		createAllocationTestTeardown(t, allocationID)
	})

	t.RunSequentiallyWithTimeout("Create allocation with invalid blobber auth ticket should fail", 10*time.Minute, func(t *test.SystemTest) {
		// Update blobber config to make restricted blobbers to true
		blobber := blobbersList[0]
		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobber.Id,
			"is_restricted": "true",
		}))
		require.Nil(t, err, strings.Join(output, "\n"))

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"is_restricted": "false",
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5"}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "not enough blobbers to honor the allocation")

		var preferredBlobbers, blobberAuthTickets string
		for i, bb := range blobbersList {
			if i == len(blobbersList)-1 {
				preferredBlobbers += bb.Id
				break
			}

			preferredBlobbers += bb.Id + ","
			if i == 0 {
				blobberAuthTickets += "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl,"
			} else {
				blobberAuthTickets += ","
			}
		}

		options = map[string]interface{}{"size": "1024", "data": "3", "parity": "3", "lock": "0.5", "preferred_blobbers": preferredBlobbers, "blobber_auth_tickets": blobberAuthTickets}
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Not enough blobbers to honor the allocation")
		require.Contains(t, output[len(output)-1], "auth ticket verification failed")
	})

	t.RunSequentially("Update allocation with add restricted blobber should succeed", func(t *test.SystemTest) {
		// setup allocation and upload a file
		allocSize := int64(64 * KB * 2)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 9,
		})

		// faucet tokens

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/dir" + filename
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)
		blobberID, err := GetBlobberIDNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)

		setupWallet(t, configPath)
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err)

		addBlobber := getBlobber(t, blobberID)
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobberID,
			"not_available": false,
			"is_restricted": false,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		addBlobberAuthTicket, err := getBlobberAuthTicket(t, blobberID, addBlobber.BaseURL, wallet.ClientID)
		require.Nil(t, err)

		t.Cleanup(func() {
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobberID,
				"not_available": false,
				"is_restricted": false,
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                blobberID,
			"add_blobber_auth_ticket":    addBlobberAuthTicket,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		assertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
		fref, err := VerifyFileRefFromBlobber(walletFile, configFile, allocationID, blobberID, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref) // not nil when the file exists
	})

	t.RunSequentially("Update allocation with replace blobber and add restricted blobber should succeed", func(t *test.SystemTest) {
		// setup allocation and upload a file
		allocSize := int64(64 * KB * 2)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 9,
		})

		// faucet tokens

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		remotePath := "/dir" + filename
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))

		wd, _ := os.Getwd()
		walletFile := filepath.Join(wd, "config", escapedTestName(t)+"_wallet.json")
		configFile := filepath.Join(wd, "config", configPath)

		blobberID, err := GetBlobberIDNotPartOfAllocation(walletFile, configFile, allocationID)
		require.Nil(t, err)
		removeBlobber, err := GetRandomBlobber(walletFile, configFile, allocationID, blobberID)
		require.Nil(t, err)

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err)

		addBlobber := getBlobber(t, blobberID)
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":    blobberID,
			"not_available": false,
			"is_restricted": false,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		addBlobberAuthTicket, err := getBlobberAuthTicket(t, blobberID, addBlobber.BaseURL, wallet.ClientID)
		require.Nil(t, err)

		t.Cleanup(func() {
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobberID,
				"not_available": false,
				"is_restricted": false,
			}))
			require.Nil(t, err, strings.Join(output, "\n"))
		})

		params := createParams(map[string]interface{}{
			"allocation":                 allocationID,
			"set_third_party_extendable": nil,
			"add_blobber":                blobberID,
			"add_blobber_auth_ticket":    addBlobberAuthTicket,
			"remove_blobber":             removeBlobber,
		})

		output, err = updateAllocation(t, configPath, params, true)
		require.Nil(t, err, "error updating allocation", strings.Join(output, "\n"))
		assertOutputMatchesAllocationRegex(t, updateAllocationRegex, output[0])
		assertOutputMatchesAllocationRegex(t, repairCompletednRegex, output[len(output)-1])
		fref, err := VerifyFileRefFromBlobber(walletFile, configFile, allocationID, blobberID, remotePath)
		require.Nil(t, err)
		require.NotNil(t, fref) // not nil when the file exists
	})
}

func getBlobberAuthTicket(t *test.SystemTest, blobberID, blobberUrl, clientID string) (string, error) {
	zboxWallet, err := getWalletForName(t, configPath, zboxTeamWallet)
	require.Nil(t, err, "could not get zbox wallet")

	var authTicket string
	signatureScheme := zcncrypto.NewSignatureScheme("bls0chain")
	_ = signatureScheme.SetPrivateKey("26e4adfa189350df06bf1983569e03a50fb69d6112386e76610e8b08cc90a009")
	_ = signatureScheme.SetPublicKey(zboxWallet.ClientPublicKey)

	signature, err := signatureScheme.Sign(hex.EncodeToString([]byte(zboxWallet.ClientPublicKey)))
	if err != nil {
		return authTicket, err
	}

	url := blobberUrl + "/v1/auth/generate?client_id=" + clientID + "&round=" + strconv.FormatInt(1000000000, 10)
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return authTicket, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Zbox-Signature", signature)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return authTicket, err
	}
	defer resp.Body.Close()
	var responseMap map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseMap)
	if err != nil {
		return "", err
	}
	authTicket = responseMap["auth_ticket"]
	if authTicket == "" {
		return "", common.NewError("500", "Error getting auth ticket from blobber")
	}

	return authTicket, nil
}
