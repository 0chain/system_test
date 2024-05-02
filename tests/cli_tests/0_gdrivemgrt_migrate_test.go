package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0Gdrive(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if gdriveAccessToken == "" {
		t.Skip("Gdrive Access Token was missing")
	}

	t.SetSmokeTests("Should migrate existing Gdrive folder and files successfully")
	println("Should migrate existing Gdrive folder and files successfully")
	t.RunSequentially("Should migrate existing Gdrive folder and files  successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": gdriveAccessToken,
			// "wallet":       escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"source":     "Gdrive",
		}))
		println(output)
		println(err)
		require.Equal(t, allocationID, allocationID)
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty folder successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": gdriveAccessToken,
			"wallet":       escapedTestName(t) + "_wallet.json",
			"allocation":   allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(5 * KB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": gdriveAccessToken,
			"allocation":   allocationID,
			"source":       "Gdrive",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error creating allocation: allocation_creation_failed: invalid request: insufficient allocation size", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when folder does not exist", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": gdriveAccessToken,
			"folder":       "undefined",
			"wallet":       escapedTestName(t) + "_wallet.json",
			"allocation":   allocationID,
			"source":       "Gdrive",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error S3: GetBucketLocation, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		_ = setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": gdriveAccessToken,
			"wallet":       escapedTestName(t) + "_wallet.json",
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access token invalid", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"access-token": "invalid",
			"wallet":       escapedTestName(t) + "_wallet.json",
			"allocation":   allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error Gdrive: Get Location, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromGdrive(t, configPath, createParams(map[string]interface{}{
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Equal(t, output[0], "Error: Gdrive credentials missing", "Output was not as expected", strings.Join(output, "\n"))
	})
}

func migrateFromGdrive(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Migrating Gdrive  to Zus...")
	t.Logf(fmt.Sprintf("params %v", params))
	t.Logf(fmt.Sprintf("cli %v", cliConfigFilename))
	return cliutils.RunCommand(t, fmt.Sprintf("./s3migration migrate  %s", params), 1, time.Hour*2)
}
