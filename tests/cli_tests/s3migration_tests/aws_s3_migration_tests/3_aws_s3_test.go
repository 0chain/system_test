package s3migration_tests

import (
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/stretchr/testify/require"
)

func Test0S3Migration(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.S3SecretKey == "" || shared.ConfigData.S3AccessKey == "" {
		t.Skip("shared.ConfigData.S3SecretKey or shared.ConfigData.S3AccessKey was missing")
	}

	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.RunSequentially("Should migrate existing bucket successfully", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate empty bucket successfully", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     "system-tests-empty",
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when bucket too large for allocation", func(t *test.SystemTest) {
		allocSize := int64(64 * shared.KB)
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     shared.DefaultWallet,
			"allocation": allocationID,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "max_allocation_size", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when bucket does not exist", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     "invalid",
			"wallet":     shared.DefaultWallet,
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error S3: GetBucketLocation, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when bucket flag missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"wallet":     shared.DefaultWallet,
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: open : no such file or directory", "Output was not as expected", strings.Join(output, "\n")) // FIXME error message could be better here
	})

	t.RunSequentially("Should fail when allocation flag missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     shared.DefaultWallet,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access key invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error S3: GetBucketLocation, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access key missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Equal(t, output[0], "Error: aws credentials missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when secret key invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": "invalid",
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error S3: GetBucketLocation, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when secret key missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Equal(t, output[0], "Error: aws credentials missing", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access and secret key invalid", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": "invalid",
			"secret-key": "invalid",
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: operation error S3: GetBucketLocation, https response error StatusCode: 403", "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should fail when access and secret key missing", func(t *test.SystemTest) {
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"bucket":     shared.ConfigData.S3BucketName,
			"wallet":     cliutils.EscapedTestName(t) + "_wallet.json",
			"allocation": shared.DefaultAllocationId,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Equal(t, output[0], "Error: aws credentials missing", "Output was not as expected", strings.Join(output, "\n"))
	})
}
