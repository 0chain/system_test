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

	allocSize := int64(50 * MB)
	test_allocationId := setupAllocation(t, configPath, map[string]interface{}{
		"size": allocSize,
	})
	test_walletName := escapedTestName(t)
	createWalletForName(test_walletName)

	t.RunSequentially("Should migrate existing Micrsoft OneDrive folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "'" + oneDriveAccessToken + "'",
			"secret-key": "'" + oneDriveRefreshToken + "'",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationId,
			"source":     "onedrive",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "'" + oneDriveAccessToken + "'",
			"secret-key": "'" + oneDriveRefreshToken + "'",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationId,
			"source":     "onedrive",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		// Prepare the parameters in the desired order using Param structs
		createWalletForName(escapedTestName(t))

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "'" + oneDriveAccessToken + "'",
			"secret-key": "'" + oneDriveRefreshToken + "'",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"source":     "onedrive",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		fmt.Printf("Output: %v\n", output)
		fmt.Printf("Error: %v\n", err)
		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when token and refresh token is invalid", func(t *test.SystemTest) {
		createWalletForName(escapedTestName(t))
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": test_allocationId,
			"source":     "onedrive",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Access token: InvalidAuthenticationToken", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"access-key": oneDriveAccessToken,
			"secret-key": oneDriveRefreshToken,
			"wallet":     test_walletName + "_wallet.json",
			"allocation": test_allocationId,
			"source":     "src",
			"config":     configPath,
			"configDir":  configDir,
			"skip":       1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
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
