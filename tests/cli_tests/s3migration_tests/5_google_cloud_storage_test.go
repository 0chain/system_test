package s3migration_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func Test0GoogleCloudStorage(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.GoogleCloudAccessToken == "" || shared.ConfigData.GoogleCloudRefreshToken == "" || shared.ConfigData.GoogleClientId == "" || shared.ConfigData.GoogleClientSecret == "" {
		t.Skip("Missing Required details for google cloud console")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Google Cloud successfully")

	t.RunSequentially("Should migrate existing Google Cloud folder and files  successfully", func(t *test.SystemTest) {
		allocationId := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    shared.ConfigData.GoogleCloudAccessToken,
			"secret-key":    shared.ConfigData.GoogleCloudRefreshToken,
			"wallet":        EscapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		totalCount, totalMigrated, err := cli_utils.GetmigratedDataID(output)

		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, totalCount, totalMigrated, "Total count of migrated files is not equal to total migrated files")
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocationId := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    shared.ConfigData.GoogleCloudAccessToken,
			"secret-key":    shared.ConfigData.GoogleCloudRefreshToken,
			"wallet":        EscapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		_, totalMigrated, err := cli_utils.GetmigratedDataID(output)

		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, 0, totalMigrated, "Total count of migrated files is not equal to total migrated files")
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    shared.ConfigData.GoogleCloudAccessToken,
			"secret-key":    shared.ConfigData.GoogleCloudRefreshToken,
			"wallet":        shared.DefaultWallet,
			"source":        "google_cloud_storage",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when client credentials is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    shared.ConfigData.GoogleCloudAccessToken,
			"secret-key":    shared.ConfigData.GoogleCloudRefreshToken,
			"wallet":        shared.DefaultWallet,
			"source":        "google_cloud_storage",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid client credentials/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when key is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    "invalid",
			"secret-key":    "invalid",
			"wallet":        shared.DefaultWallet,
			"source":        "google_cloud_storage",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Access key", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"client-id":     shared.ConfigData.GoogleClientId,
			"client-secret": shared.ConfigData.GoogleClientSecret,
			"access-key":    shared.ConfigData.GoogleCloudAccessToken,
			"secret-key":    shared.ConfigData.GoogleCloudRefreshToken,
			"wallet":        shared.DefaultWallet,
			"source":        "invalid",
			"config":        shared.ConfigPath,
			"configDir":     shared.ConfigDir,
			"skip":          0,
			"wd":            "0chainmigration",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})
}
