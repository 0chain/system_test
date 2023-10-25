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

	t.RunSequentiallyWithTimeout("Resume upload should work fine", 10*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(2 * GB)
		fileSize := int64(500 * MB)

		output, err := executeFaucetWithTokens(t, configPath, 100.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
			"lock": 50,
		})

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
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

		output, err = uploadFile(t, configPath, map[string]interface{}{
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
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])
	})
}
