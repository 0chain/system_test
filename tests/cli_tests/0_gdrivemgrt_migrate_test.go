package cli_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0Gdrive(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if gdriveAccessToken == "" {
		t.Skip("Gdrive Access Token was missing")
	}

	t.SetSmokeTests("Should migrate existing Gdrive folder and files successfully")

	t.RunSequentially("Should migrate existing Gdrive folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": gdriveAccessToken,
			"secret-key": gdriveRefreshToken,
			"allocation": allocationID,
			"source":     "google_drive",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"config":     configPath,
			"configDir":  configDir,
		}))
		require.GreaterOrEqual(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": gdriveAccessToken,
			"secret-key": gdriveRefreshToken,
			"allocation": allocationID,
			"source":     "google_drive",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"config":     configPath,
			"configDir":  configDir,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder does not exist", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": gdriveAccessToken,
			"secret-key": gdriveRefreshToken,
			"allocation": allocationID,
			"source":     "google_drive",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"config":     configPath,
			"configDir":  configDir,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		_ = setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": gdriveAccessToken,
			"secret-key": gdriveRefreshToken,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "google_drive",
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

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "google_drive",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid Credentials", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "google_drive",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		t.Logf("EXpected log  %v", strings.Join(output, "\n"))
		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: access key, secret key")
	})
}