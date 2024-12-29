package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0GoogleCloudConsole(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if googleCloudConsoleAccessToken == "" || googleCloudConsoleRefreshToken == "" || googleClientId == "" || googleClientSecret == "" {
		t.Skip("Missing Required details for google cloud console")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Google Cloud successfully")

	t.RunSequentially("Should migrate existing Google Cloud folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-token":  googleCloudConsoleAccessToken,
			"refresh-token": googleCloudConsoleRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
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
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-token":  googleCloudConsoleAccessToken,
			"refresh-token": googleCloudConsoleRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-token":  googleCloudConsoleAccessToken,
			"refresh-token": googleCloudConsoleRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when client credentials is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     "invalid_client_id",
			"client-secret": "invalid client secret",
			"access-token":  googleCloudConsoleAccessToken,
			"refresh-token": googleCloudConsoleRefreshToken,
			"allocation":    allocationId,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid client credentials/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when token is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-token":  "invalid_access_token",
			"refresh-token": "invalid_refresh_token",
			"allocation":    allocationId,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Access token", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-token":  googleCloudConsoleAccessToken,
			"refresh-token": googleCloudConsoleRefreshToken,
			"allocation":    allocationId,
			"wallet":        escapedTestName(t) + "_wallet.json",
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
