package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestResumeUpload(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	t.RunSequentiallyWithTimeout("Resume upload should work fine", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2 * GB)
		fileSize := int64(500 * MB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 50,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		param := map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/",
			"localpath":   filename,
			"chunknumber": 500, // 64KB * 500 = 32M
		}
		upload_param := createParams(param)
		command := fmt.Sprintf(
			"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
			upload_param,
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		cmd, _ := cliutils.StartCommandWithoutRetry(command)
		uploaded := waitPartialUploadAndInterrupt(t, cmd)
		t.Logf("the uploaded is %v ", uploaded)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/",
			"localpath":   filename,
			"chunknumber": 500, // 64KB * 500 = 32M
		}, false)

		require.Nil(t, err, strings.Join(output, "\n"))
		pattern := `(\d+ / \d+)\s+(\d+\.\d+%)`
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(output[0], -1)
		require.GreaterOrEqual(t, len(matches), 1)
		a := matches[len(matches)-1]
		first, err := strconv.ParseInt(strings.Fields(a)[0], 10, 64)
		require.Nil(t, err, "error in extracting size from output, adjust the regex")
		second, err := strconv.ParseInt(strings.Fields(a)[2], 10, 64)
		require.Nil(t, err, "error in extracting size from output, adjust the regex")
		require.Less(t, first, second) // Ensures upload didn't start from beginning
		require.Len(t, output, 2)
		expected := fmt.Sprintf(
			"Status completed callback. Type = text/plain. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})

	t.RunSequentiallyWithTimeout("Resume upload with same filename having same filesize with diff content(Negative)", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2 * GB)
		fileSize := int64(300 * MB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 50,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		param := map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 20,
		}
		upload_param := createParams(param)
		command := fmt.Sprintf(
			"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
			upload_param,
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		cmd, _ := cliutils.StartCommandWithoutRetry(command)
		uploaded := waitPartialUploadAndInterrupt(t, cmd)
		t.Logf("the uploaded is %v ", uploaded)

		//  creating file with samename & size but with new content
		_ = createFileWithSize(filename, fileSize)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 20,
		}, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		//  asserting output
		require.Contains(t, output[1], "file_store_error: Error committing to file store. hash_mismatch:")
		require.Error(t, err)
		////asserting error
		expected:= "exit status 1"
		
		require.Equal(t, expected, err.Error())
	})

	t.RunSequentiallyWithTimeout("Resume upload with diff filename having same filesize (Negative)", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2 * GB)
		fileSize := int64(100 * MB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 50,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		param := map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 500, // 64KB * 500 = 32M
		}
		upload_param := createParams(param)
		command := fmt.Sprintf(
			"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
			upload_param,
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		cmd, _ := cliutils.StartCommandWithoutRetry(command)
		uploaded := waitPartialUploadAndInterrupt(t, cmd)
		t.Logf("the uploaded is %v ", uploaded)

		//  renaming the same file
		filename = generateRandomTestFileName(t)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 500, // 64KB * 500 = 32M
		}, false)

		require.NotNil(t, err, strings.Join(output, "\n"))
		//  asserting output
		require.Contains(t, output[0], "no such file or directory")
		require.Error(t, err)
		//  asserting error
		expected := fmt.Sprintf(
			"exit status 1",
		)
		require.Equal(t, expected, err.Error())
	})

	t.RunSequentiallyWithTimeout("Should discard previous progress and treat as new upload when file size if different", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2 * GB)
		fileSize := int64(500 * MB)
		fileSize2 := int64(550 * MB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 50,
		})

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		param := map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 20, // 64KB * 500 = 32M
		}
		upload_param := createParams(param)
		command := fmt.Sprintf(
			"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
			upload_param,
			escapedTestName(t)+"_wallet.json",
			configPath,
		)

		cmd, _ := cliutils.StartCommandWithoutRetry(command)
		uploaded := waitPartialUploadAndInterrupt(t, cmd)
		t.Logf("the uploaded is %v ", uploaded)

		// increasing the file size to test the negative flow
		_ = createFileWithSize(filename, fileSize2)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  "/dummy",
			"localpath":   filename,
			"chunknumber": 20, // 64KB * 500 = 32M
		}, false)

		//  asserting positive output
		require.Nil(t, err, strings.Join(output, "\n"))
		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = dummy",
		)
		require.Equal(t, expected, output[1])
	})
}
