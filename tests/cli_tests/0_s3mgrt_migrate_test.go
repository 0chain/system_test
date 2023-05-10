package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func Test0S3Migration(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}
	t.Parallel()
	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate existing bucket successfully", func(t *test.SystemTest) {
		allocSize := int64(1 * GB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		println("output length: ", len(output))
		fmt.Printf("output: %v", output)
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n")) //FIXME: There should be an code of 1 on failure but it is always zero
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})
}

func migrateFromS3(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Migrating S3 bucket to Zus...")
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./s3mgrt migrate --silent --access-key %s --secret-key %s --bucket %s --wallet %s --configDir ./config --config %s --network %s %s",
		s3AccessKey, s3SecretKey, s3bucketName, escapedTestName(t)+"_wallet.json", cliConfigFilename, cliConfigFilename, params))
}
