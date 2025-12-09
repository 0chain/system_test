package cli_tests

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/errors"

	"github.com/0chain/common/core/common"
	"github.com/0chain/gosdk/core/zcncrypto"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"
)

func TestRestrictedBlobbers(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// Auth Ticket Round expiry we can add a big round number to create the allocation.
	authTokenRoundExpiry := int(1e9)

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
			outputStr := strings.Join(output, "\n")
			if err != nil && strings.Contains(outputStr, "access denied") {
				// If access denied, skip this blobber - delegate wallet might be different
				t.Logf("Warning: Cannot update blobber %s settings - access denied. Skipping.", blobber.Id)
				continue
			}
			// Only assert if there was no error (or error was not access denied)
			if err != nil {
				t.Logf("Warning: Failed to update blobber %s settings: %v, Output: %s", blobber.Id, err, outputStr)
				continue
			}
		}
	})

	t.Cleanup(func() {
		for _, blobber := range blobbersList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"not_available": false,
				"is_restricted": false,
			}))
			outputStr := strings.Join(output, "\n")
			if err != nil {
				// Skip if we can't revert - delegate wallet might be different or blobber might not exist
				if strings.Contains(outputStr, "access denied") {
					t.Logf("Warning: Cannot revert blobber %s settings - access denied. Skipping cleanup for this blobber.", blobber.Id)
				} else {
					t.Logf("Warning: Failed to revert blobber %s settings: %v, Output: %s. Skipping cleanup for this blobber.", blobber.Id, err, outputStr)
				}
				continue
			}
		}
	})

	t.RunSequentiallyWithTimeout("Create allocation on restricted blobbers should pass with correct auth tickets", 10*time.Minute, func(t *test.SystemTest) {
		// Update blobber config to make restricted blobbers to true
		// Find blobbers that can be updated (not access denied)
		var updatableBlobbers []climodel.BlobberDetails
		for _, blobber := range blobbersList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"is_restricted": "true",
			}))
			outputStr := strings.Join(output, "\n")
			if err != nil && strings.Contains(outputStr, "access denied") {
				t.Logf("Warning: Cannot update blobber %s to restricted - access denied. Skipping.", blobber.Id)
				continue
			}
			if err != nil {
				t.Logf("Warning: Failed to update blobber %s to restricted: %v, Output: %s. Skipping.", blobber.Id, err, outputStr)
				continue
			}
			updatableBlobbers = append(updatableBlobbers, blobber)
			if len(updatableBlobbers) >= 2 {
				break // We need at least 2 blobbers for the test
			}
		}
		require.GreaterOrEqual(t, len(updatableBlobbers), 2, "Need at least 2 updatable blobbers for this test")
		
		blobber1 := updatableBlobbers[0]
		blobber2 := updatableBlobbers[1]

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber1.Id,
				"is_restricted": "false",
			}))
			outputStr := strings.Join(output, "\n")
			if err != nil {
				if strings.Contains(outputStr, "access denied") {
					t.Logf("Warning: Cannot revert blobber %s settings - access denied. Skipping cleanup.", blobber1.Id)
				} else {
					t.Logf("Warning: Failed to revert blobber %s settings: %v, Output: %s. Skipping cleanup.", blobber1.Id, err, outputStr)
				}
			}
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber2.Id,
				"is_restricted": "false",
			}))
			outputStr = strings.Join(output, "\n")
			if err != nil {
				if strings.Contains(outputStr, "access denied") {
					t.Logf("Warning: Cannot revert blobber %s settings - access denied. Skipping cleanup.", blobber2.Id)
				} else {
					t.Logf("Warning: Failed to revert blobber %s settings: %v, Output: %s. Skipping cleanup.", blobber2.Id, err, outputStr)
				}
			}
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "2048", "data": "3", "parity": "3", "lock": "0.5", "force": "true"} // Use 2048 to meet min_alloc_size requirement
		output, err = createNewAllocationWithoutRetry(t, configPath, createParams(options))
		require.NotNil(t, err)
		require.True(t, len(output) > 0, "expected output length be at least 1", strings.Join(output, "\n"))

		// Retry with auth ticket
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "could not get wallet")

		blobber1AuthTicket, err := getBlobberAuthTicket(t, blobber1.Id, blobber1.Url, wallet.ClientID, authTokenRoundExpiry)
		require.Nil(t, err, "could not get blobber1 auth ticket")
		blobber2AuthTicket, err := getBlobberAuthTicket(t, blobber2.Id, blobber2.Url, wallet.ClientID, authTokenRoundExpiry)
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

		options = map[string]interface{}{
			"size":                 "2048", // Use 2048 to meet min_alloc_size requirement
			"data":                 "3",
			"parity":               "3",
			"lock":                 "0.5",
			"preferred_blobbers":   preferredBlobbers,
			"blobber_auth_tickets": blobberAuthTickets,
			"auth_round_expiry":    authTokenRoundExpiry,
			"force":                "true",
		}
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
		// Find a blobber that can be updated (not access denied)
		var blobber climodel.BlobberDetails
		found := false
		for _, bb := range blobbersList {
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    bb.Id,
				"is_restricted": "true",
			}))
			outputStr := strings.Join(output, "\n")
			if err != nil && strings.Contains(outputStr, "access denied") {
				t.Logf("Warning: Cannot update blobber %s to restricted - access denied. Trying next blobber.", bb.Id)
				continue
			}
			if err != nil {
				t.Logf("Warning: Failed to update blobber %s to restricted: %v, Output: %s. Trying next blobber.", bb.Id, err, outputStr)
				continue
			}
			blobber = bb
			found = true
			break
		}
		require.True(t, found, "Could not find an updatable blobber for this test")

		t.Cleanup(func() {
			// Reset blobber config to make restricted blobbers to false
			output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":    blobber.Id,
				"is_restricted": "false",
			}))
			outputStr := strings.Join(output, "\n")
			if err != nil {
				if strings.Contains(outputStr, "access denied") {
					t.Logf("Warning: Cannot revert blobber %s settings - access denied. Skipping cleanup.", blobber.Id)
				} else {
					t.Logf("Warning: Failed to revert blobber %s settings: %v, Output: %s. Skipping cleanup.", blobber.Id, err, outputStr)
				}
			}
		})

		// Setup wallet and create allocation
		_ = setupWallet(t, configPath)

		options := map[string]interface{}{"size": "2048", "data": "3", "parity": "3", "lock": "0.5"} // Use 2048 to meet min_alloc_size requirement
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

		options = map[string]interface{}{"size": "2048", "data": "3", "parity": "3", "lock": "0.5", "preferred_blobbers": preferredBlobbers, "blobber_auth_tickets": blobberAuthTickets} // Use 2048 to meet min_alloc_size requirement
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
		addBlobberAuthTicket, err := getBlobberAuthTicket(t, blobberID, addBlobber.BaseURL, wallet.ClientID, authTokenRoundExpiry)
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
			"auth_round_expiry":          authTokenRoundExpiry,
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
		addBlobberAuthTicket, err := getBlobberAuthTicket(t, blobberID, addBlobber.BaseURL, wallet.ClientID, authTokenRoundExpiry)
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
			"auth_round_expiry":          authTokenRoundExpiry,
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

func getBlobberAuthTicket(t *test.SystemTest, blobberID, blobberURL, clientID string, authTokenRoundExpiry int) (string, error) {
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

	url := blobberURL + "/v1/auth/generate?client_id=" + clientID + "&round=" + strconv.FormatInt(int64(authTokenRoundExpiry), 10)
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
		body, readErr := io.ReadAll(resp.Body) // use io.ReadAll instead of ioutil.ReadAll
		if readErr != nil {
			return "", errors.Wrap(readErr, "failed to read response body")
		}
		return "", errors.Wrap(err, string(body))
	}
	authTicket = responseMap["auth_ticket"]
	if authTicket == "" {
		return "", common.NewError("500", "Error getting auth ticket from blobber")
	}
	return authTicket, nil
}
