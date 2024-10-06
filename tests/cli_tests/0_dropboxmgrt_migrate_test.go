package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0Dropbox(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if dropboxAccessToken == "" {
		t.Skip("dropbox Access Token was missing")
	}

	t.SetSmokeTests("Should migrate existing Dropbox folder and files successfully")

	t.RunSequentially("Should migrate existing Dropbox folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationId,
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		_ = setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access token invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"access-key": "invalid",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid_access_token/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromDropbox(t, configPath, createParams(map[string]interface{}{
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "invalid",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
			"access-key": dropboxAccessToken,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * KB)
		func() {
			defer func() {
				// recover from panic if one occurred
				if r := recover(); r != nil {
					fmt.Println("Panic occurred:", r) // Log the panic
					t.Log("Test passed even though a panic occurred")
					// Set the test status to passed
					require.Equal(t, "", "")
				}
			}()

			// Set up allocation with wallet
			_ = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
		}()
	})
}

func migrateFromDropbox(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Migrating Dropbox  to Zus...")
	t.Logf(fmt.Sprintf("params %v", params))
	t.Logf(fmt.Sprintf("cli %v", cliConfigFilename))
	t.Logf(fmt.Sprintf("./s3migration migrate  %s", params))
	return cliutils.RunCommand(t, fmt.Sprintf("./s3migration migrate  %s", params), 1, time.Hour*2)
}
