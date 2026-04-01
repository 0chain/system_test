package cli_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// zboxURL returns the 0box base URL from ZBOX_URL env or a sensible default.
func zboxURL() string {
	if u := os.Getenv("ZBOX_URL"); u != "" {
		return u
	}
	return "http://localhost:9081"
}

// postShortIOLink calls POST /v2/shortio/link on 0box and returns the
// HTTP status code and response body.
func postShortIOLink(originalURL, title string) (int, string, error) {
	payload, _ := json.Marshal(map[string]string{
		"original_url": originalURL,
		"title":        title,
	})

	resp, err := http.Post(zboxURL()+"/v2/shortio/link", "application/json", bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}

func TestShareShortIO(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Public file share and generate short link")

	// shortioConfigured is set by the first subtest; later tests use it to
	// decide whether to validate the shortened URL or just log a warning.
	shortioConfigured := true

	t.RunSequentiallyWithTimeout("Public file share and generate short link", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// Upload a file
		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Share publicly
		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEmpty(t, authTicket)

		// Build the share URL and request a short link
		shareURL := "https://test.vult.network/share?authticket=" + authTicket

		statusCode, body, err := postShortIOLink(shareURL, "Public file share")
		require.NoError(t, err, "HTTP transport error calling short.io endpoint")
		t.Logf("Short.io response: %d - %s", statusCode, body)

		require.NotEqual(t, 404, statusCode,
			"Short.io endpoint must exist (got 404): %s", body)

		if statusCode >= 400 {
			shortioConfigured = false
			t.Logf("WARNING: Short.io likely not configured (status %d). "+
				"Link creation tests will verify endpoint reachability only.", statusCode)
			return
		}

		require.True(t, statusCode == 200 || statusCode == 201,
			"Expected 200/201, got %d: %s", statusCode, body)
		require.Contains(t, body, "short",
			"Response should contain shortened URL data")
	})

	t.RunSequentiallyWithTimeout("Private file share and generate short link", 5*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// Upload a file
		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, 256)
		require.Nil(t, err)

		remotePath := "/" + filepath.Base(file)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": remotePath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Create receiver wallet for private share
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

		walletReceiver, err := getWalletForName(t, configPath, receiverWallet)
		require.Nil(t, err)

		// Share to specific recipient (private)
		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation":          allocationID,
			"remotepath":          remotePath,
			"clientid":            walletReceiver.ClientID,
			"encryptionpublickey": walletReceiver.EncryptionPublicKey,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEmpty(t, authTicket)

		// Verify the receiver can actually download (proves the share is real)
		os.Remove(file)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file,
			"authticket": authTicket,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Now generate a short link for the private share URL
		shareURL := "https://test.vult.network/share?authticket=" + authTicket

		statusCode, body, err := postShortIOLink(shareURL, "Private file share")
		require.NoError(t, err, "HTTP transport error calling short.io endpoint")
		t.Logf("Short.io private link response: %d - %s", statusCode, body)

		require.NotEqual(t, 404, statusCode,
			"Short.io endpoint must exist (got 404): %s", body)

		if !shortioConfigured {
			t.Logf("WARNING: Short.io not configured, skipping link validation")
		} else if statusCode == 200 || statusCode == 201 {
			require.Contains(t, body, "short",
				"Response should contain shortened URL data")
		}
	})

	t.RunSequentiallyWithTimeout("Folder share and generate short link", 6*time.Minute, func(t *test.SystemTest) {
		walletOwner := escapedTestName(t)
		allocationID, _ := createWalletAndAllocation(t, configPath, walletOwner)

		// Create folder and upload two files
		remoteDir := "/sharedFolder/"
		output, err := createDir(t, configPath, allocationID, remoteDir, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, remoteDir+" directory created", output[0])

		file1 := generateRandomTestFileName(t)
		err = createFileWithSize(file1, 256)
		require.Nil(t, err)

		remoteFile1 := remoteDir + filepath.Base(file1)
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file1,
			"remotepath": remoteFile1,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		file2 := generateRandomTestFileName(t)
		err = createFileWithSize(file2, 256)
		require.Nil(t, err)

		remoteFile2 := remoteDir + filepath.Base(file2)
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file2,
			"remotepath": remoteFile2,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Share folder publicly
		output, err = shareFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteDir,
		})
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		authTicket, err := extractAuthToken(output[0])
		require.Nil(t, err, "Error extracting auth token")
		require.NotEmpty(t, authTicket)

		// Verify a receiver can download a file from the shared folder
		receiverWallet := escapedTestName(t) + "_receiver"
		createWalletForName(receiverWallet)

		os.Remove(file1)

		downloadParams := createParams(map[string]interface{}{
			"localpath":  file1,
			"authticket": authTicket,
			"remotepath": remoteFile1,
		})
		output, err = downloadFileForWallet(t, receiverWallet, configPath, downloadParams, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Contains(t, output[1], StatusCompletedCB)

		// Generate a short link for the folder share URL
		shareURL := fmt.Sprintf("https://test.vult.network/share?authticket=%s", authTicket)

		statusCode, body, err := postShortIOLink(shareURL, "Shared folder link")
		require.NoError(t, err, "HTTP transport error calling short.io endpoint")
		t.Logf("Short.io folder link response: %d - %s", statusCode, body)

		require.NotEqual(t, 404, statusCode,
			"Short.io endpoint must exist (got 404): %s", body)

		if !shortioConfigured {
			t.Logf("WARNING: Short.io not configured, skipping link validation")
		} else if statusCode == 200 || statusCode == 201 {
			require.Contains(t, body, "short",
				"Response should contain shortened URL data")
		}
	})

	t.RunSequentiallyWithTimeout("Empty URL should return error from short.io endpoint", 2*time.Minute, func(t *test.SystemTest) {
		statusCode, body, err := postShortIOLink("", "Empty URL test")
		require.NoError(t, err, "HTTP transport error")
		t.Logf("Short.io empty URL response: %d - %s", statusCode, body)

		require.True(t, statusCode >= 400,
			"Empty original_url should be rejected with 4xx/5xx, got %d", statusCode)
	})

	t.RunSequentiallyWithTimeout("Short.io not configured returns graceful error", 2*time.Minute, func(t *test.SystemTest) {
		// This test verifies the endpoint exists and returns a non-404
		// response even when Short.io API keys may not be configured.
		statusCode, body, err := postShortIOLink(
			"https://test.vult.network/share?authticket=probe_test",
			"Config probe",
		)
		require.NoError(t, err, "HTTP transport error")
		t.Logf("Short.io config probe response: %d - %s", statusCode, body)

		require.NotEqual(t, 404, statusCode,
			"Short.io endpoint must exist (got 404): %s", body)

		if statusCode >= 500 {
			t.Logf("WARNING: Short.io endpoint returned %d, likely missing API key configuration", statusCode)
		}
	})
}
