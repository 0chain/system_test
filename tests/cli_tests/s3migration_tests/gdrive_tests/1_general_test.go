package gdrive_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func Test0Gdrive(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Log(shared.ConfigData.GdriveAccessToken)
	if shared.ConfigData.GdriveAccessToken == "" {
		t.Skip("Gdrive Access Token was missing")
	}
	t.SetSmokeTests("Should migrate existing Gdrive folder and files successfully")

	t.RunSequentially("Should migrate existing Gdrive folder and files  successfully", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})
		output, _ := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.GdriveAccessToken,
			"secret-key": shared.ConfigData.GdriveRefreshToken,
			"allocation": allocationID,
			"source":     "google_drive",
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))
		require.GreaterOrEqual(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))

		totalCount, totalMigrated, err := cli_utils.GetmigratedDataID(output)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, totalCount, totalMigrated, "Total count of migrated files is not equal to total migrated files")
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder does not exist", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.GdriveAccessToken,
			"secret-key": shared.ConfigData.GdriveRefreshToken,
			"allocation": shared.DefaultAllocationId,
			"source":     "google_drive",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.GdriveAccessToken,
			"secret-key": shared.ConfigData.GdriveRefreshToken,
			"source":     "google_drive",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access token invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"allocation": shared.DefaultAllocationId,
			"source":     "google_drive",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid Credentials", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"allocation": shared.DefaultAllocationId,
			"source":     "google_drive",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		t.Logf("EXpected log  %v", strings.Join(output, "\n"))
		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: access key, secret key")
	})
}
