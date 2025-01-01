package cli_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0Dropbox(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if dropboxAccessToken == "" || dropboxRefreshToken == "" {
		t.Skip("Missing Required Tokens for Dropbox migration")
	}

	t.SetSmokeTests("Should migrate existing Dropbox folder and files successfully")

	t.RunSequentially("Should migrate existing Dropbox folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"secret-key": dropboxRefreshToken,
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

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"secret-key": dropboxRefreshToken,
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

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"secret-key": dropboxRefreshToken,
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

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
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

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "dropbox",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		t.Logf("EXpected log  %v", strings.Join(output, "\n"))
		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: access key, secret key")
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": dropboxAccessToken,
			"secret-key": dropboxRefreshToken,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "invalid",
			"config":     configPath,
			"configDir":  configDir,
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source. Supported sources: s3, google_drive, dropbox, onedrive, azure, google_cloud_storage")
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * KB)
		var err error
		defer func() {
			require.Contains(t, err.Error(), "allocation match not found")
		}()
		_, err = setupAllocationWithWalletWithoutTest(t, escapedTestName(t)+"_wallet.json", configPath, map[string]interface{}{
			"size": allocSize,
		})
	})
}

func setupAllocationWithWalletWithoutTest(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) (string, error) {
	options := map[string]interface{}{"size": "10000000", "lock": "5"}

	for _, params := range extraParams {
		for k, v := range params {
			options[k] = v
		}
	}
	createWalletForName(walletName)
	output, _ := createNewAllocationForWallet(t, walletName, cliConfigFilename, createParams(options))
	defer func() {
		fmt.Printf("err: %v\n", output)
	}()
	return getAllocationID(output[0])
}
