package s3migration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func Test0Dropbox(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.DropboxAccessToken == "" || shared.ConfigData.DropboxRefreshToken == "" {
		t.Skip("Missing Required Tokens for Dropbox migration")
	}

	t.SetSmokeTests("Should migrate existing Dropbox folder and files successfully")


	t.RunSequentially("Should migrate existing Dropbox folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * shared.MB)
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath,  map[string]interface{}{
			"size": allocSize,
		})
		output, _ := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.DropboxAccessToken,
			"secret-key": shared.ConfigData.DropboxRefreshToken,
			"allocation": allocationID,
			"source":     "dropbox",
			"wallet":     EscapedTestName(t) + "_wallet.json",
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip": 0,
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.DropboxAccessToken,
			"secret-key": shared.ConfigData.DropboxRefreshToken,
			"allocation": shared.DefaultAllocationId,
			"source":     "dropbox",
			"wallet":     shared.DefaultAllocationId,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip": 0,
		}))


		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.DropboxAccessToken,
			"secret-key": shared.ConfigData.DropboxRefreshToken,
			"source":     "dropbox",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip": 0,
		}))


		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access token invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"allocation": shared.DefaultAllocationId,
			"source":     "dropbox",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip": 0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid_access_token/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, cli_utils.CreateParams(map[string]interface{}{
			"allocation": shared.DefaultAllocationId,
			"source":     "dropbox",
			"wallet":     shared.DefaultWallet,
			"config":     shared.ConfigPath,
			"configDir":  shared.ConfigDir,
			"skip": 0,
		}))
		
		t.Logf("EXpected log  %v", strings.Join(output, "\n"))
		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: access key, secret key")
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * shared.KB)
		var err error
		defer func() {
			require.Contains(t, err.Error(), "allocation match not found")
		}()
		_, err = setupAllocationWithWalletWithoutTest(t, EscapedTestName(t)+"_wallet.json", shared.ConfigPath, map[string]interface{}{
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
	cli_utils.CreateWalletForName(shared.RootPath, walletName)
	output, _ := cli_utils.CreateNewAllocationForWallet(t, walletName, cliConfigFilename,shared.RootPath,  cli_utils.CreateParams(options))
	defer func() {
		fmt.Printf("err: %v\n", output)
	}()
	return cli_utils.GetAllocationID(output[0])
}
