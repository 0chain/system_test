package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

/*
Tests in here are skipped until the feature has been fixed
*/
func Test___FlakyBrokenScenarios(t *testing.T) {
	balance := 0.8 // 800.000 mZCN
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Parallel()

	t.Run("Send with description", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Send ZCN with description is temporarily broken due to json object enforcement")
		}
		t.Parallel()

		targetWallet := escapedTestName(t) + "_TARGET"

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		target, err := getWalletForName(t, configPath, targetWallet)
		require.Nil(t, err, "Error occurred when retrieving target wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		output, err = sendZCN(t, configPath, target.ClientID, "1", "rich description", true)
		require.Nil(t, err, "Unexpected send failure", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "Send tokens success", output[0])
		// cannot verify transaction payload at this moment due to transaction hash not being printed.
	})

	t.Run("Tokens should move from write pool balance to challenge pool acc. to expected upload cost", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Blobber write pool balance is not being updated correctly")
		}
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		initialWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilon(t, 0.8, intToZCN(initialWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId)
		require.Less(t, 0, len(initialWritePool[0].Blobber))
		require.Equal(t, true, initialWritePool[0].Locked)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, 1024*5)
		require.Nil(t, err, "error while generating file: ", err)

		// Get expected upload cost
		output, _ = getUploadCostInUnit(t, configPath, allocationID, filename)

		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		// Expected cost is given in "per 720 hours", we need 1 hour
		// Expected cost takes into account data+parity, so we divide by that
		actualExpectedUploadCostInZCN := (expectedUploadCostInZCN / ((2 + 2) * 720))

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
		})

		// Get the new Write-Pool info after upload
		output, err = writePoolInfo(t, configPath, true)
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Nil(t, err, "error fetching write pool info", strings.Join(output, "\n"))

		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, 0.8, intToZCN(finalWritePool[0].Balance), epsilon)
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, finalWritePool[0].AllocationId)
		require.Less(t, 0, len(finalWritePool[0].Blobber))
		require.Equal(t, true, finalWritePool[0].Locked)

		// Get Challenge-Pool info after upload
		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

		challengePool := climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("([a-f0-9]{64}):challengepool:%s", allocationID)), challengePool.Id)
		require.IsType(t, int64(1), challengePool.StartTime)
		require.IsType(t, int64(1), challengePool.Expiration)
		require.IsType(t, int64(1), challengePool.Balance)
		require.False(t, challengePool.Finalized)

		// Blobber pool balance should reduce by (write price*filesize) for each blobber
		totalChangeInWritePool := float64(0)
		for i := 0; i < len(finalWritePool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalWritePool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalWritePool[0].Blobber[i].Balance)

			// deduce tokens
			diff := intToZCN(initialWritePool[0].Blobber[i].Balance) - intToZCN(finalWritePool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] write pool has decreased by [%v] tokens after upload", i, diff)
			totalChangeInWritePool += diff
		}

		require.InEpsilon(t, actualExpectedUploadCostInZCN, totalChangeInWritePool, epsilon, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", actualExpectedUploadCostInZCN, totalChangeInWritePool)
		require.InEpsilon(t, totalChangeInWritePool, intToZCN(challengePool.Balance), epsilon, "expected challenge pool balance to match deducted amount from write pool [%v] but balance was actually [%v]", totalChangeInWritePool, intToZCN(challengePool.Balance))
	})

	t.Run("Each blobber's read pool balance should reduce by download cost", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Blobber read pool balance is not being updated correctly")
		}
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   balance,
			"size":   10485760,
			"expire": "1h",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		path, err := filepath.Abs("tmp")
		require.Nil(t, err)

		filename := cliutils.RandomAlphaNumericString(10) + "_test.txt"
		fullPath := fmt.Sprintf("%s/%s", path, filename)
		err = createFileWithSize(fullPath, 1024*5)
		require.Nil(t, err, "error while generating file: ", err)

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fullPath,
			"remotepath": "/",
		})

		// Lock read pool tokens
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"tokens":     0.4,
			"duration":   "5m",
		})
		output, err = readPoolLock(t, configPath, params, true)
		require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Equal(t, "locked", output[0])

		// Read pool before download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		initialReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &initialReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Id)
		require.InEpsilon(t, 0.4, intToZCN(initialReadPool[0].Balance), epsilon, "read pool balance did not match expected")
		require.IsType(t, int64(1), initialReadPool[0].ExpireAt)
		require.Equal(t, allocationID, initialReadPool[0].AllocationId)
		require.Less(t, 0, len(initialReadPool[0].Blobber))
		require.Equal(t, true, initialReadPool[0].Locked)

		for i := 0; i < len(initialReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), initialReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), initialReadPool[0].Blobber[i].Balance)
			t.Logf("Blobber [%v] balance is [%v]", i, intToZCN(initialReadPool[0].Blobber[i].Balance))
		}

		output, err = getDownloadCostInUnit(t, configPath, allocationID, "/"+filename)
		require.Nil(t, err, "Could not get download cost", strings.Join(output, "\n"))

		expectedDownloadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))

		unit := strings.Fields(output[0])[1]
		expectedDownloadCostInZCN = unitToZCN(expectedDownloadCostInZCN, unit)

		// Download the file

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/" + filename,
			"localpath":  "../../internal/dummy_file/five_MB_test_file_dowloaded",
		}), true)
		require.Nil(t, err, "Downloading the file failed", strings.Join(output, "\n"))

		defer os.Remove("../../internal/dummy_file/five_MB_test_file_dowloaded")

		require.Len(t, output, 2)
		require.Equal(t, "Status completed callback. Type = application/octet-stream. Name = "+filename, output[1])

		// Read pool after download
		output, err = readPoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Error fetching read pool", strings.Join(output, "\n"))

		finalReadPool := []climodel.ReadPoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalReadPool)
		require.Nil(t, err, "Error unmarshalling read pool", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Id)
		require.Less(t, intToZCN(finalReadPool[0].Balance), 0.4)
		require.IsType(t, int64(1), finalReadPool[0].ExpireAt)
		require.Equal(t, allocationID, finalReadPool[0].AllocationId)
		require.Equal(t, len(initialReadPool[0].Blobber), len(finalReadPool[0].Blobber))
		require.True(t, finalReadPool[0].Locked)

		for i := 0; i < len(finalReadPool[0].Blobber); i++ {
			require.Regexp(t, regexp.MustCompile("([a-f0-9]{64})"), finalReadPool[0].Blobber[i].BlobberID)
			require.IsType(t, int64(1), finalReadPool[0].Blobber[i].Balance)

			// amount deducted
			diff := intToZCN(initialReadPool[0].Blobber[i].Balance) - intToZCN(finalReadPool[0].Blobber[i].Balance)
			t.Logf("blobber [%v] read pool was deducted by [%v]", i, diff)
			require.InEpsilon(t, expectedDownloadCostInZCN, diff, epsilon, "blobber [%v] read pool was deducted by [%v] rather than the expected [%v]", i, diff, expectedDownloadCostInZCN)
		}
	})

	t.Run("update file with thumbnail", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Downloading thumbnail is not working")
		}
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		localFilePath := generateFileAndUpload(t, allocationID, remotepath, filesize)

		thumbnailFile := updateFileWithThumbnailURL(t, "https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png", allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		output, err := downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		defer func() {
			// Delete the downloaded thumbnail file
			err = os.Remove(thumbnailFile)
			require.Nil(t, err)
		}()
		createAllocationTestTeardown(t, allocationID)
	})

	t.Run("update thumbnail of uploaded file", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Downloading thumbnail is not working")
		}
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN. Default allocation has 2 data shards and 2 parity shards
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"size":   10 * MB,
			"tokens": 2,
		})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		thumbnail := "upload_thumbnail_test.png"
		output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("wget %s -O %s", "https://en.wikipedia.org/static/images/project-logos/enwiki-2x.png", thumbnail))
		require.Nil(t, err, "Failed to download thumbnail png file: ", strings.Join(output, "\n"))

		localFilePath := generateFileAndUploadWithParam(t, allocationID, remotepath, filesize, map[string]interface{}{"thumbnailpath": thumbnail})

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		err = os.Remove(thumbnail)
		require.Nil(t, err)

		// Update with new thumbnail
		thumbnail = updateFileWithThumbnailURL(t, "https://icons-for-free.com/iconfiles/png/512/eps+file+format+png+file+icon-1320167140989998942.png", allocationID, "/"+filepath.Base(localFilePath), localFilePath, int64(filesize))

		output, err = downloadFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath + filepath.Base(localFilePath),
			"localpath":  "tmp/",
			"thumbnail":  true,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		err = os.Remove(thumbnail)
		require.Nil(t, err)

		createAllocationTestTeardown(t, allocationID)
	})
}
