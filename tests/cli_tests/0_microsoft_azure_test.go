package cli_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0MicrosoftAzure(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if accountName == "" {
		t.Skip("Missing Account name for required for migration")
	}

	if connectionString == "" {
		t.Skip("Missing Connection String required for migration")
	}

	t.SetSmokeTests("Should migrate existing files and folder from Microsoft Azure Container successfully")

	allocSize := int64(50 * MB)
	test_allocationId := setupAllocation(t, configPath, map[string]interface{}{
		"size": allocSize,
	})
	test_walletName := escapedTestName(t)
	createWalletForName(test_walletName)

	t.RunSequentially("Should migrate existing Microsoft Azure folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		t.Log("container name", containerName)

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name":      accountName,
			"container":         containerName,
			"wallet":            escapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": "'" + connectionString + "'",
		}))

		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name":      accountName,
			"container":         containerName,
			"wallet":            escapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": "'" + connectionString + "'",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name":      accountName,
			"container":         containerName,
			"wallet":            test_walletName + "_wallet.json",
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": connectionString,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when connection string is invalid", func(t *test.SystemTest) {

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"connection-string": "invalid",
			"account-name":      accountName,
			"container":         containerName,
			"wallet":            test_walletName + "_wallet.json",
			"allocation":        test_allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), " connection string is either blank or malformed", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when connection string is missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name": accountName,
			"container":    containerName,
			"wallet":       test_walletName + "_wallet.json",
			"allocation":   test_allocationId,
			"source":       "azure",
			"config":       configPath,
			"configDir":    configDir,
			"skip":         1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Missing fields: connection string", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name": accountName,
			"wallet":       test_walletName + "_wallet.json",
			"allocation":   test_allocationId,
			"source":       "invalid",
			"config":       configPath,
			"configDir":    configDir,
			"skip":         1,
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
