package onedrive_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func TestOneDrive(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.OneDriveAccessToken == "" || shared.ConfigData.OneDriveRefreshToken == "" {
		t.Skip("Missing Required details for OneDrive migration")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Microsoft OneDrive successfully")

	t.RunSequentially("Should migrate existing Micrsoft OneDrive folder and files  successfully", func(t *test.SystemTest) {
		allocationId := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, _ := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "'" + shared.ConfigData.OneDriveAccessToken + "'",
			"secret-key": "'" + shared.ConfigData.OneDriveRefreshToken + "'",
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationId,
			"source":     "onedrive",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "'" + shared.ConfigData.OneDriveAccessToken + "'",
			"secret-key": "'" + shared.ConfigData.OneDriveRefreshToken + "'",
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
			"source":     "onedrive",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "'" + shared.ConfigData.OneDriveAccessToken + "'",
			"secret-key": "'" + shared.ConfigData.OneDriveRefreshToken + "'",
			"wallet":     shared.DefaultWallet,
			"source":     "onedrive",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       1,
		}))

		fmt.Printf("Output: %v\n", output)
		fmt.Printf("Error: %v\n", err)
		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when token and refresh token is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"wallet":     shared.DefaultWallet,
			"allocation": shared.DefaultAllocationId,
			"source":     "onedrive",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip":       0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Access token: InvalidAuthenticationToken", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "'" + shared.ConfigData.OneDriveAccessToken + "'",
			"secret-key": "'" + shared.ConfigData.OneDriveRefreshToken + "'",
			"source":     "invalid",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"wallet":     shared.DefaultWallet,
			"allocation": shared.DefaultAllocationId,
			"skip":       1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * shared.KB)
		var err error
		defer func() {
			require.Contains(t, err.Error(), "allocation match not found")
		}()
		_, err = shared.SetupAllocationWithWalletWithoutTest(t, cli_utils.EscapedTestName(t)+"_wallet.json", shared.ConfigPath, map[string]interface{}{
			"size": allocSize,
		})
	})
}
