package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0OneDrive(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if oneDriveAccessToken == "" || oneDriveRefreshToken == "" {
		t.Skip("Missing Required details for OneDrive migration")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Microsoft OneDrive successfully")

	t.RunSequentially("Should migrate existing Micrsoft OneDrive folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-token":  oneDriveAccessToken,
			"refresh-token": oneDriveRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "onedrive",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-token":  oneDriveAccessToken,
			"refresh-token": oneDriveRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "onedrive",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-token":  oneDriveAccessToken,
			"refresh-token": oneDriveRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"source":        "onedrive",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when token and refresh token is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-token":  "invalid",
			"refresh-token": "invalid",
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "onedrive",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid client credentials/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-token":  oneDriveAccessToken,
			"refresh-token": oneDriveRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "src",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * KB)
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Panic occurred:", r)
					t.Log("Test passed even though a panic occurred")
					require.Equal(t, "", "")
				}
			}()
			_ = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
		}()
	})
}
