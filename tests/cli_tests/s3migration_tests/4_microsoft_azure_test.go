package s3migration_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func Test0MicrosoftAzure(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.AccountName == "" {
		t.Skip("Missing Account name for required for migration")
	}

	if shared.ConfigData.ConnectionString == "" {
		t.Skip("Missing Connection String required for migration")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Microsoft Azure Container successfully")

	t.RunSequentially("Should migrate existing Microsoft Azure folder and files  successfully", func(t *test.SystemTest) {
		allocationId := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, _ := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name":      shared.ConfigData.AccountName,
			"container":         shared.ConfigData.ContainerName,
			"connection-string": "'" + shared.ConfigData.ConnectionString + "'",
			"wallet":            cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            shared.ConfigPath,
			"configDir":         shared.ConfigDir,
			"skip":              0,
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

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name":      shared.ConfigData.AccountName,
			"container":         shared.ConfigData.ContainerName,
			"connection-string": "'" + shared.ConfigData.ConnectionString + "'",
			"wallet":            cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            shared.ConfigPath,
			"configDir":         shared.ConfigDir,
			"skip":              0,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name":      shared.ConfigData.AccountName,
			"container":         shared.ConfigData.ContainerName,
			"connection-string": "'" + shared.ConfigData.ConnectionString + "'",
			"wallet":            shared.DefaultWallet,
			"source":            "azure",
			"config":            shared.ConfigPath,
			"configDir":         shared.ConfigDir,
			"skip":              0,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when connection string is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name":      shared.ConfigData.AccountName,
			"container":         shared.ConfigData.ContainerName,
			"connection-string": "invalid",
			"wallet":            shared.DefaultWallet,
			"allocation":        shared.DefaultAllocationId,
			"source":            "azure",
			"config":            shared.ConfigPath,
			"configDir":         shared.ConfigDir,
			"skip":              0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), " connection string is either blank or malformed", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when connection string is missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name": shared.ConfigData.AccountName,
			"container":    shared.ConfigData.ContainerName,
			"wallet":       shared.DefaultWallet,
			"allocation":   shared.DefaultAllocationId,
			"source":       "azure",
			"config":       shared.ConfigPath,
			"configDir":    shared.ConfigDir,
			"skip":         0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: connection string", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"account-name":      shared.ConfigData.AccountName,
			"container":         shared.ConfigData.ContainerName,
			"connection-string": "'" + shared.ConfigData.ConnectionString + "'",
			"wallet":            shared.DefaultWallet,
			"allocation":        shared.DefaultAllocationId,
			"source":            "invalid",
			"config":            shared.ConfigPath,
			"configDir":         shared.ConfigDir,
			"skip":              0,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid source", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		size := int64(5 * shared.KB)
		var err error
		defer func() {
			require.Contains(t, err.Error(), "allocation match not found")
		}()
		_, err = setupAllocationWithWalletWithoutTest(t, cli_utils.EscapedTestName(t)+"_wallet.json", shared.ConfigPath, map[string]interface{}{
			"size": size,
		})
	})
}
