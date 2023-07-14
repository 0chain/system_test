package cli_tests

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
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

	t.RunWithTimeout("Resume upload should work", 5*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(600 * MB)
		fileSize := int64(500 * MB)

		for i := 0; i < 6; i++ {
			output, err := executeFaucetWithTokens(t, configPath, 9.0)
			require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))
		}

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"lock":   50,
			"expire": "30m",
		})
		defer func() {
			createAllocationTestTeardown(t, allocationID)
		}()

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, fileSize)
		require.Nil(t, err)
		defer func() {
			os.Remove(filename) //nolint: errcheck
		}()

		cmd := startUploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, false)
		out, err := cmd.StdoutPipe()
		require.Nil(t, err, "Failed to create stdout pipe")

		err = cmd.Start()
		require.Nil(t, err, "Upload failed to start")

		// Create a new scanner and read the output line by line.
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println(line)
			t.Log(line)

			re := regexp.MustCompile(`\s+(\d+(\.\d+)?)%`)
			match := re.FindStringSubmatch(line)

			if len(match) > 1 {
				percent, err := strconv.ParseFloat(match[1], 64)
				require.Nil(t, err, "Failed to parse percent")

				// If the output line indicates a certain amount of the upload is done, you could stop it.
				if percent > 20.0 {
					err = cmd.Process.Signal(os.Interrupt)
					require.Nil(t, err, "Failed to send interrupt signal")
					t.Log("Partial upload successful, upload has been interrupted")
					break
				}
			}
		}
		t.Log("Passed first upload")

		// Allow command to stop
		time.Sleep(5 * time.Second)

		// Resume upload
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
}

// returns stdoutPipe that can be used to read line by line
func startUploadFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) *exec.Cmd {
	return startUploadFileForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func startUploadFileForWallet(t *test.SystemTest, wallet, cliConfigFilename string, param map[string]interface{}, retry bool) *exec.Cmd {
	t.Logf("Uploading file...")

	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox upload %s --wallet %s_wallet.json --configDir ./config --config %s",
		p,
		wallet,
		cliConfigFilename,
	)
	return cliutils.Command(cmd)
}
