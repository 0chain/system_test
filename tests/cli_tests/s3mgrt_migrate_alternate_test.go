package cli_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

/*
Let's list down the cases
delete source is not covered let's check
--dup_suffix
how to verify --encrypt option I think you need to see the code
--migrate to how to create that remote path
-- how to verify older than, just create older than 10 min file
-- resume  what do you mean by previous state
-- retry count , how to verify it, don't you think that retry would be one in ideal case so how to increase the retry count for testing purpose
-- skip can be tested
-- but what is duplicate
-- what is this wd
*/
func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}
	t.Parallel()
	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate as copy bucket successfully ", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		output, err = migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.Run("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migarate-to": "/root2",
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.Run("Should migrate existing bucket successfully with encryption on", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

}
