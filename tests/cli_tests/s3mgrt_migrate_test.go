package cli_tests

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestS3Migration(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	t.RunSequentiallyWithTimeout("Should migrate existing bucket successfully", 15*time.Minute, func(t *test.SystemTest) {
		allocSize := int64(1 * GB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
		}))
		require.Nil(t, err, "Unexpected migration failure %s", strings.Join(output, "\n")) //FIXME: Exit code of zero on failure
		require.Greater(t, output, 1, "No output was returned.")
		require.Equal(t, "Migration successful", output[0])
	})
}

func migrateFromS3(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Logf("Migrating S3 bucket to Zus...")
	return cliutils.RunCommandWithoutRetry(fmt.Sprintf(
		"./s3mgrt migrate --silent --access-key %s --secret-key %s --bucket %s --wallet %s --configDir ./config --config %s %s",
		s3AccessKey, s3SecretKey, s3bucketName, escapedTestName(t)+"_wallet.json", cliConfigFilename, params))
}
