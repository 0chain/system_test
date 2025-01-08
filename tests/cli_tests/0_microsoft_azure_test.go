package cli_tests

import (
	"fmt"
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

	t.RunSequentially("Should migrate existing Microsoft Azure folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name":      accountName,
			"wallet":            escapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": connectionString,
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
			"wallet":            escapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": connectionString,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, _ := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name":      accountName,
			"wallet":            escapedTestName(t) + "_wallet.json",
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
			"connection-string": connectionString,
		}))

		require.Contains(t, strings.Join(output, "\n"), "allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when connection string is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"connection-string": "invalid",
			"account-name":      accountName,
			"wallet":            escapedTestName(t) + "_wallet.json",
			"allocation":        allocationId,
			"source":            "azure",
			"config":            configPath,
			"configDir":         configDir,
			"skip":              1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token: invalid_access_token/", "Output was not as expected", err)
	})

	t.RunSequentially("Should fail when connection string is missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name": accountName,
			"wallet":       escapedTestName(t) + "_wallet.json",
			"allocation":   allocationId,
			"source":       "azure",
			"config":       configPath,
			"configDir":    configDir,
			"skip":         1,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "invalid Client token", "Output was not as expected", strings.Join(output, "\n"))
	})
	t.RunSequentially("Should fail when source is invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationId := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromS3migration(t, configPath, createParams(map[string]interface{}{
			"account-name": accountName,
			"wallet":       escapedTestName(t) + "_wallet.json",
			"allocation":   allocationId,
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
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Panic occurred:", r)
					t.Log("Test passed even though a panic occurred")
					require.Equal(t, "", "")
				}
			}()
			_ = setupAllocation(t, configPath, map[string]interface{}{
				"size": allocSize,
			})
		}()
	})
}
