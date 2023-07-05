package cli_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestUpload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Upload File With half Size of the Allocation Should Work")

	t.Parallel()

	// Success Scenarios

	t.Run("Upload File With half Size of the Allocation Should Work", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload multiple files less than size of the Allocation Should Work", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(256 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		for i := 0; i < 2; i++ {
			filename := generateRandomTestFileName(t)
			err := createFileWithSize(filename, fileSize)
			require.Nil(t, err)

			output, err := uploadFile(t, configPath, map[string]interface{}{
				"allocation": allocationID,
				"remotepath": "/",
				"localpath":  filename,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2)

			expected := fmt.Sprintf(
				"Status completed callback. Type = application/octet-stream. Name = %s",
				filepath.Base(filename),
			)
			require.Equal(t, expected, output[1])
		}
	})

	t.Run("Upload File to Root Directory Should Work", func(t *test.SystemTest) { // todo: slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.RunWithTimeout("Upload file concurrently to root directory, should work", 6*time.Minute, func(t *test.SystemTest) { // todo: slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		const remotePathPrefix = "/"

		var fileNames [2]string

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(currentIndex int) {
				defer wg.Done()

				fileName := generateRandomTestFileName(t)
				err := createFileWithSize(fileName, fileSize)
				require.Nil(t, err)

				fileNameBase := filepath.Base(fileName)

				fileNames[currentIndex] = fileNameBase

				op, err := uploadFile(t, configPath, map[string]interface{}{
					"allocation": allocationID,
					"remotepath": path.Join(remotePathPrefix, fileNameBase),
					"localpath":  fileName,
				}, true)

				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(i)
		}
		wg.Wait()

		const expectedPattern = "Status completed callback. Type = application/octet-stream. Name = %s"

		for i := 0; i < 2; i++ {
			require.Nil(t, errorList[i], strings.Join(outputList[i], "\n"))
			require.Len(t, outputList[i], 2, strings.Join(outputList[i], "\n"))
			require.Equal(t, fmt.Sprintf(expectedPattern, fileNames[i]), outputList[i][1], "Output is not appropriate")
		}
	})

	t.Run("Upload File to a Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/" + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.RunWithTimeout("Upload File to a Directory without Filename Should Work", 60*time.Second, func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := "Status completed callback. Type = application/octet-stream. Name = " + filepath.Base(filename)
		require.Equal(t, expected, output[1])

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/dir/",
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var listResults []climodel.ListFileResult
		err = json.Unmarshal([]byte(output[0]), &listResults)
		require.Nil(t, err, "Decoding list results failed\n", strings.Join(output, "\n"))

		require.Len(t, listResults, 1)
		result := listResults[0]

		require.Equal(t, filepath.Base(filename), result.Name)
		require.Equal(t, "/dir/"+filepath.Base(filename), result.Path)
		require.Equal(t, fileSize, result.ActualSize)
		require.Equal(t, "f", result.Type)
		require.Equal(t, "", result.EncryptionKey)
	})

	t.Run("Upload File to Nested Directory Should Work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/nested/dir/" + filepath.Base(filename),
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File with Thumbnail Should Work", func(t *test.SystemTest) {
		allocSize := int64(10 * 1024 * 1024)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		thumbnail := escapedTestName(t) + "thumbnail.png"

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":    allocationID,
			"remotepath":    "/",
			"localpath":     filename,
			"thumbnailpath": thumbnail,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload Image File Should Work", func(t *test.SystemTest) {
		allocSize := int64(10 * 1024 * 1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := escapedTestName(t) + "image.png"
		//nolint
		fileBytes, _ := base64.StdEncoding.DecodeString(`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`)
		err := os.WriteFile(filename, fileBytes, os.ModePerm)
		require.Nil(t, err, "failed to generate thumbnail", err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = image/png. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.RunWithTimeout("Upload Video File Should Work", 2*time.Minute, func(t *test.SystemTest) { //todo: slow
		allocSize := int64(400 * 1024 * 1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"tokens": 9,
			"expire": "10m",
		})

		output, err := cliutils.RunCommand(t, "wget http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4 -O test_video.mp4", 3, 2*time.Second)
		require.Nil(t, err, "Failed to download test video file: ", strings.Join(output, "\n"))

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  "./test_video.mp4",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := "Status completed callback. Type = video/mp4. Name = test_video.mp4"
		require.Equal(t, expected, output[1])
	})

	t.RunWithTimeout("Upload Large File Should Work", 6*time.Minute, func(t *test.SystemTest) { // todo: this is slow, see https://0chain.slack.com/archives/G014PQ61WNT/p1669672933550459
		allocSize := int64(2 * GB)
		fileSize := int64(1 * GB)

		for i := 0; i < 6; i++ {
			output, err := executeFaucetWithTokens(t, configPath, 9.0)
			require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))
		}

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"lock":   50,
			"expire": "30m",
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/",
			"localpath":   filename,
			"chunknumber": 1024, // 64KB * 1024 = 64M
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Upload File with Encryption Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, 10)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
			"encrypt":    "",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.Run("Data shards do not require more allocation space", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   2,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 1, "Output length was less than expected")
		require.True(t, strings.HasPrefix(output[len(output)-1], "Status completed callback"), "Expected success string to be present")
	})

	// Failure Scenarios

	//FIXME: the CLI could check allocation size before attempting an upload to save wasted time/bandwidth
	t.Run("Upload File too large - file size larger than allocation should fail", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(2 * MB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, strings.Contains(strings.Join(output, "\n"), "alloc: no enough space left in allocation"), strings.Join(output, "\n"))
	})

	t.Run("Upload File too large - parity shards take up allocation space - more than half Size of the Allocation Should Fail when 1 parity shard", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(513 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, strings.Contains(strings.Join(output, "\n"), "upload_failed"), strings.Join(output, "\n"))
	})

	t.Run("Upload File too large - parity shards take up allocation space - more than quarter Size of the Allocation Should Fail when 3 parity shards", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(257 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 3,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))

		require.True(t, strings.Contains(strings.Join(output, ""), "upload_failed"), strings.Join(output, "\n"))
	})

	t.Run("Upload File to Existing File Should Fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(1024)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Upload the file again to same directory
		output, err = uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t, strings.Contains(strings.Join(output, ""), "upload_failed"), strings.Join(output, "\n"))
	})

	t.Run("Upload File to Non-Existent Allocation Should Fail", func(t *test.SystemTest) {
		fileSize := int64(256)

		_, err := createWallet(t, configPath)
		require.Nil(t, err)

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": "ab12mn34as90",
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := "Error fetching the allocation. allocation_fetch_error: " +
			"Error fetching the allocation.internal_error: can't get allocation: error retrieving allocation: ab12mn34as90, error: record not found"
		require.Equal(t, expected, output[0])
	})

	t.Run("Upload File to Other's Allocation Should Fail", func(t *test.SystemTest) {
		var otherAllocationID string

		allocSize := int64(2048)
		fileSize := int64(256)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		t.Run("Get Other Allocation ID", func(t *test.SystemTest) {
			otherAllocationID = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
		})

		// Upload using allocationID: should work
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		// Upload using otherAllocationID: should not work
		output, err = uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": otherAllocationID,
			"remotepath": "/",
			"localpath":  filename,
		})

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.True(t,
			strings.Contains(strings.Join(output, ""), "upload_failed"), strings.Join(output, "\n"))
	})

	t.Run("Upload Non-Existent File Should Fail", func(t *test.SystemTest) {
		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		filename := "non-existent-file.txt"

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  "non-existent-file.txt",
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		expected := fmt.Sprintf(
			"Upload failed. open %s: no such file or directory",
			filename,
		)
		require.Equal(t, expected, output[0])
	})

	t.Run("Upload Blank File Should Fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(0)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		})
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Upload failed. EOF", output[0])
	})

	t.Run("Upload without any Parameter Should Fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = uploadFileWithoutRetry(t, configPath, nil)

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("Upload to Allocation without remotepath and authticket Should Fail", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 2048,
		})

		output, err := uploadFileWithoutRetry(t, configPath, map[string]interface{}{
			"allocation": allocationID,
		})

		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})

	t.Run("Upload File longer than 100 chars should fail", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		dirPath := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))
		randomFilename := cliutils.RandomAlphaNumericString(101)
		filename := fmt.Sprintf("%s%s%s_test.txt", dirPath, string(os.PathSeparator), randomFilename)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		require.NotNil(t, err, "error uploading file")
		require.Len(t, output, 1)
		require.Contains(t, output[0], "filename is longer than 100 characters")
	})

	t.Run("Upload File should fail if upload file option is forbidden", func(t *test.SystemTest) {
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":          allocSize,
			"forbid_upload": nil,
		})

		dirPath := strings.TrimSuffix(os.TempDir(), string(os.PathSeparator))
		randomFilename := cliutils.RandomAlphaNumericString(101)
		filename := fmt.Sprintf("%s%s%s_test.txt", dirPath, string(os.PathSeparator), randomFilename)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		require.NotNil(t, err)
		require.Len(t, output, 1)
		require.Contains(t, output[0], "this options for this file is not permitted for this allocation")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
		}), false)
		require.Nil(t, err)
		require.NotContains(t, output[0], filename)
	})

	t.RunWithTimeout("Tokens should move from write pool balance to challenge pool acc. to expected upload cost", 10*time.Minute, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "Failed to execute faucet transaction", strings.Join(output, "\n"))

		allocParam := createParams(map[string]interface{}{
			"lock":   0.8,
			"size":   10485760,
			"expire": "10m",
		})
		output, err = createNewAllocation(t, configPath, allocParam)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
		require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

		allocationID := strings.Fields(output[0])[2]

		// Write pool balance should increment to 1
		initialAllocation := getAllocation(t, allocationID)
		require.Equal(t, 0.8, intToZCN(initialAllocation.WritePool))

		// Get Challenge-Pool info after upload
		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

		challengePool := climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, 1024*1024*0.5)
		require.Nil(t, err, "error while generating file: ", err)

		// record time in minute
		startTime := float64(time.Now().Second())

		// upload a dummy 5 MB file
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
		})

		// record time in minute
		endTime := float64(time.Now().Second())

		minutesElapsed := endTime - startTime

		minutesElapsed = 10 - minutesElapsed/60

		output, _ = getUploadCostInUnit(t, configPath, allocationID, filename)
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		unit := strings.Fields(output[0])[1]
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)

		finalAllocation := getAllocation(t, allocationID)

		// Get Challenge-Pool info after upload
		output, err = challengePoolInfo(t, configPath, allocationID)
		require.Nil(t, err, "Could not fetch challenge pool", strings.Join(output, "\n"))

		challengePool = climodel.ChallengePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &challengePool)
		require.Nil(t, err, "Error unmarshalling challenge pool info", strings.Join(output, "\n"))

		require.Regexp(t, regexp.MustCompile(fmt.Sprintf("([a-f0-9]{64}):challengepool:%s", allocationID)), challengePool.Id)
		require.IsType(t, int64(1), challengePool.StartTime)
		require.IsType(t, int64(1), challengePool.Expiration)
		require.IsType(t, int64(1), challengePool.Balance)
		require.False(t, challengePool.Finalized)

		totalChangeInWritePool := intToZCN(initialAllocation.WritePool - finalAllocation.WritePool)

		require.InEpsilon(t, expectedUploadCostInZCN, totalChangeInWritePool, 0.15, "expected write pool balance to decrease by [%v] but has actually decreased by [%v]", expectedUploadCostInZCN, totalChangeInWritePool)
		require.Equal(t, totalChangeInWritePool, intToZCN(challengePool.Balance), "expected challenge pool balance to match deducted amount from write pool [%v] but balance was actually [%v]", totalChangeInWritePool, intToZCN(challengePool.Balance))
	})

	sampleVideos := [][]string{
		{
			"https://filesamples.com/samples/video/wtv/sample_960x400_ocean_with_audio.wtv",
			"test_wtv_video",
			"wtv",
		},
		{
			"https://filesamples.com/samples/video/mts/sample_960x400_ocean_with_audio.mts",
			"test_mts_video",
			"mts",
		},
		{
			"https://filesamples.com/samples/video/f4v/sample_960x400_ocean_with_audio.f4v",
			"test_f4v_video",
			"f4v",
		},
		{
			"https://filesamples.com/samples/video/flv/sample_960x400_ocean_with_audio.flv",
			"test_flv_video",
			"flv",
		},
		{
			"https://filesamples.com/samples/video/3gp/sample_960x400_ocean_with_audio.3gp",
			"test_3gp_video",
			"3gp",
		},
		{
			"https://filesamples.com/samples/video/m4v/sample_960x400_ocean_with_audio.m4v",
			"test_m4v_video",
			"m4v",
		},
		{
			"https://filesamples.com/samples/video/mov/sample_960x400_ocean_with_audio.mov",
			"test_mov_video",
			"mov",
		},
		{
			"https://filesamples.com/samples/video/mp4/sample_960x400_ocean_with_audio.mp4",
			"test_mp4_video",
			"mp4",
		},
		{
			"https://filesamples.com/samples/video/mjpeg/sample_960x400_ocean_with_audio.mjpeg",
			"test_mjpeg_video",
			"mjpeg",
		},
		{
			"https://filesamples.com/samples/video/mkv/sample_960x400_ocean_with_audio.mkv",
			"test_mkv_video",
			"mkv",
		},
		{
			"https://filesamples.com/samples/video/hevc/sample_960x400_ocean_with_audio.hevc",
			"test_hevc_video",
			"hevc",
		},
		{
			"https://filesamples.com/samples/video/m2ts/sample_960x400_ocean_with_audio.m2ts",
			"test_m2ts_video",
			"m2ts",
		},
		{
			"https://filesamples.com/samples/video/m2v/sample_960x400_ocean_with_audio.m2v",
			"test_m2v_video",
			"m2v",
		},
		{
			"https://filesamples.com/samples/video/mpeg/sample_960x400_ocean_with_audio.mpeg",
			"test_mpeg_video",
			"mpeg",
		},
		{
			"https://filesamples.com/samples/video/mpg/sample_960x400_ocean_with_audio.mpg",
			"test_mpg_video",
			"mpg",
		},
		{
			"https://filesamples.com/samples/video/mxf/sample_960x400_ocean_with_audio.mxf",
			"test_mxf_video",
			"mxf",
		},
		{
			"https://filesamples.com/samples/video/ogv/sample_960x400_ocean_with_audio.ogv",
			"test_ogv_video",
			"ogv",
		},
		{
			"https://filesamples.com/samples/video/rm/sample_960x400_ocean_with_audio.rm",
			"test_rm_video",
			"rm",
		},
		{
			"https://filesamples.com/samples/video/ts/sample_960x400_ocean_with_audio.ts",
			"test_ts_video",
			"ts",
		},
		{
			"https://filesamples.com/samples/video/vob/sample_960x400_ocean_with_audio.vob",
			"test_vob_video",
			"vob",
		},
		{
			"https://filesamples.com/samples/video/asf/sample_960x400_ocean_with_audio.asf",
			"test_asf_video",
			"asf",
		},
		{
			"https://filesamples.com/samples/video/avi/sample_960x400_ocean_with_audio.avi",
			"test_avi_video",
			"avi",
		},
		{
			"https://filesamples.com/samples/video/webm/sample_960x400_ocean_with_audio.webm",
			"test_webm_video",
			"webm",
		},
		{
			"https://filesamples.com/samples/video/wmv/sample_960x400_ocean_with_audio.wmv",
			"test_wmv_video",
			"wmv",
		},
	}
	for _, sampleVideo := range sampleVideos {
		videoLink := sampleVideo[0]
		videoName := sampleVideo[1]
		videoFormat := sampleVideo[2]
		t.RunWithTimeout("Upload Video File "+videoFormat+" With Web Streaming Should Work", 2*time.Minute, func(t *test.SystemTest) {
			allocSize := int64(400 * 1024 * 1024)
			allocationID := setupAllocation(t, configPath, map[string]interface{}{
				"size":   allocSize,
				"tokens": 9,
				"expire": "10m",
			})
			downloadVideo := "wget " + videoLink + " -O " + videoName + "." + videoFormat
			output, err := cliutils.RunCommand(t, downloadVideo, 3, 2*time.Second)
			require.Nil(t, err, "Failed to download test video file: ", strings.Join(output, "\n"))

			output, err = uploadFile(t, configPath, map[string]interface{}{
				"allocation":    allocationID,
				"remotepath":    "/",
				"localpath":     "./" + videoName + "." + videoFormat,
				"web-streaming": "",
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2)
			expected := "Status completed callback. Type = video/fmp4. Name = raw." + videoName + ".mp4"
			require.Equal(t, expected, output[1])
		})
	}
}

func uploadWithParam(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}) {
	uploadWithParamForWallet(t, escapedTestName(t), cliConfigFilename, param)
}
func uploadWithParamForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}) {
	filename, ok := param["localpath"].(string)
	require.True(t, ok)

	output, err := uploadFileForWallet(t, wallet, cliConfigFilename, param, true)
	require.Nil(t, err, "Upload file failed due to error ", err, strings.Join(output, "\n"))

	require.Len(t, output, 2)

	aggregatedOutput := strings.Join(output, " ")
	require.Contains(t, aggregatedOutput, StatusCompletedCB)
	require.Contains(t, aggregatedOutput, filepath.Base(filename))
}

func uploadFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return uploadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func uploadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Uploading file...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*40)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func uploadFileWithoutRetry(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}) ([]string, error) {
	t.Logf("Uploading file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		p,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func generateFileAndUpload(t *test.SystemTest, allocationID, remotepath string, size int64) string {
	return generateFileAndUploadForWallet(t, escapedTestName(t), allocationID, remotepath, size)
}

func generateFileAndUploadForWallet(t *test.SystemTest, wallet, allocationID, remotepath string, size int64) string {
	filename := generateRandomTestFileName(t)

	err := createFileWithSize(filename, size)
	require.Nil(t, err)

	// Upload parameters
	// log command with allocation id, filename and remotepath
	t.Logf("Uploading file %s to allocation %s with remotepath %s", filename, allocationID, remotepath+filepath.Base(filename))
	uploadWithParamForWallet(t, wallet, configPath, map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	})

	return filename
}

func generateFileAndUploadWithParam(t *test.SystemTest, allocationID, remotepath string, size int64, params map[string]interface{}) string {
	filename := generateRandomTestFileName(t)

	err := createFileWithSize(filename, size)
	require.Nil(t, err)

	p := map[string]interface{}{
		"allocation": allocationID,
		"localpath":  filename,
		"remotepath": remotepath + filepath.Base(filename),
	}

	for k, v := range params {
		p[k] = v
	}

	// Upload parameters
	uploadWithParam(t, configPath, p)

	return filename
}
