package cli_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0GoogleCloudStorage(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if googleCloudStorageAccessToken == "" || googleCloudStorageRefreshToken == "" || googleClientId == "" || googleClientSecret == "" {
		t.Skip("Missing Required details for google cloud console")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Google Cloud successfully")

	t.RunSequentially("Should migrate existing Google Cloud folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		test_walletName := escapedTestName(t)
		createWalletForName(test_walletName)

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-key":  googleCloudStorageAccessToken,
			"secret-key": googleCloudStorageRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",
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
			"access-key":  googleCloudStorageAccessToken,
			"secret-key": googleCloudStorageRefreshToken,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationId,
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	allocSize := int64(50 * MB)
	test_allocationId := setupAllocation(t, configPath, map[string]interface{}{
		"size": allocSize,
	})
	test_walletName := escapedTestName(t)
	createWalletForName(test_walletName)

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-key":    googleCloudStorageAccessToken,
			"secret-key":    googleCloudStorageRefreshToken,
			"wallet":        test_walletName + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when client credentials is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     "invalid_client_id",
			"client-secret": "invalid client secret",
			"access-key":    googleCloudStorageAccessToken,
			"secret-key":    googleCloudStorageRefreshToken,
			"allocation":    test_allocationId,
			"wallet":        test_walletName + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid client credentials/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when key is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-key":    "invalid_access_key",
			"secret-key":    "invalid_refresh_key",
			"allocation":    test_allocationId,
			"wallet":        test_walletName + "_wallet.json",
			"source":        "google_cloud_storage",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Access key", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"client-id":     googleClientId,
			"client-secret": googleClientSecret,
			"access-key":    googleCloudStorageAccessToken,
			"secret-key":    googleCloudStorageRefreshToken,
			"allocation":    test_allocationId,
			"wallet":        test_walletName + "_wallet.json",
			"source":        "src",
			"config":        configPath,
			"configDir":     configDir,
			"skip":          1,
			"wd": "0chainmigration",

		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})
}
