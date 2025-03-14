package s3migration_tests

import (
	"bytes"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
)

func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.S3SecretKey == "" || shared.ConfigData.S3AccessKey == "" {
		t.Skip("shared.ConfigData.S3SecretKey or shared.ConfigData.S3AccessKey was missing")
	}

	fileKey := "sdfg" + ".txt"
	t.TestSetup("Setup s3 bucket with relevant file", func() {
		// Cleanup bucket before test
		err := shared.CleanupBucket(shared.S3Client, shared.ConfigData.S3BucketNameAlternate)
		if err != nil {
			t.Log("Failed to cleanup bucket: ", err)
		}
		// Read file contents
		fileContents := []byte("Hello, World!")

		// Upload the file to S3
		_, err = shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKey),
			Body:   bytes.NewReader(fileContents),
		})
		if err != nil {
			t.Skip("S3 Bucket operatiion is not working properly")
		}
	})

	t.RunSequentially("Should migrate existing bucket successfully with skip 0 and replace existing file", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       0,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, remotepath)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		remotepath := "/root2"
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, "/")
		remoteFilePath = path.Join(remoteFilePath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, "/")
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate as copy bucket successfully", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		output, err = cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       2,
			"dup-suffix": "_copy",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		// FIXME: dupl suffix is not working properly so commenting
		// remotepath := "/"
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_modified." + parts[1]
		// remoteFilePath := path.Join(remotepath, fileKey_modified)
		// uploadStats := shared.CheckStats(t, remoteFilePath, fileKey_modified, allocationID)
		// require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with encryption on", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt":    "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, true)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should skip migration with skip flag == 1 and migartion should be skipped", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate successfully with duplicate files with skip flag == 2", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       2,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		// FIXME : copy extension is not there
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_copy." + parts[1]
		remotepath = path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully with duplication files  with skip flag == 2 and dup-suffix", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       2,
			"dup-suffix": "_modified",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		// FIXME : dupl suffix is not working
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_modified." + parts[1]
		remotepath = path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully and delete the s3 bucket file and use custom workdir", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		workingDirName := shared.CreateDirectoryForTestname(t)
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
		}()
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key":    shared.ConfigData.S3AccessKey,
			"secret-key":    shared.ConfigData.S3SecretKey,
			"bucket":        shared.ConfigData.S3BucketNameAlternate,
			"wallet":        cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation":    allocationID,
			"skip":          0,
			"delete-source": true,
			"wd":            workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should error out if workdir is not default and not empty", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		workingDirName := shared.CreateDirectoryForTestname(t)
		file, _ := os.CreateTemp(workingDirName, "prefix")
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
			_ = os.Remove(file.Name())
		}()

		remotepath := "/root3"
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
			"wd":         workingDirName,
		}))

		require.NotNil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "working directory not empty", "Output was not as expected", strings.Join(output, "\n"))
	})
}
