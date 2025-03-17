package s3migration_tests

import (
	"bytes" // #nosec
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
)

func Test0S3MigrationAlternatePart2(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if shared.ConfigData.S3SecretKey == "" || shared.ConfigData.S3AccessKey == "" {
		t.Skip("shared.ConfigData.S3SecretKey or shared.ConfigData.S3AccessKey was missing")
	}

	fileKey := "OneMinNew" + ".txt"
	t.TestSetup("Setup s3 bucket with relevant file", func() {
		// Cleanup before test
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

	t.RunSequentially("Should migrate existing bucket successfully with concurrency==4 and working dir current dir", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		workingDirName := shared.CreateDirectoryForTestname(t)
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
		}()
		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key":  shared.ConfigData.S3AccessKey,
			"secret-key":  shared.ConfigData.S3SecretKey,
			"bucket":      shared.ConfigData.S3BucketNameAlternate,
			"wallet":      cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation":  allocationID,
			"concurrency": 4,
			"wd":          workingDirName,
		}))

		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with newer than prefix", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		fileKeyNew := "oneMinOld" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"newer-than": time.Now().Unix() - 60, // start timestamp
		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with older than prefix", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		olderThanFileKey := "olderThanFile" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(olderThanFileKey),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		cliutils.Wait(t, 70*time.Second)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"older-than": time.Now().Unix() - 60, // end timestamp
		}))

		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, olderThanFileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, olderThanFileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with files staring with given prefix", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		fileKeyToBemigrated := "mgrt" + ".txt"
		fileKeyNotToBeMigrated := "noMgrt" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyToBemigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)
		_, err = shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNotToBeMigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"prefix":     "mgrt",
		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePathPos := path.Join(remoteFilePath, fileKeyToBemigrated)
		remoteFilePathNeg := path.Join(remoteFilePath, fileKeyNotToBeMigrated)
		uploadStats := shared.CheckStats(t, remoteFilePathPos, fileKeyToBemigrated, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
		uploadStats = shared.CheckStats(t, remoteFilePathNeg, fileKeyNotToBeMigrated, allocationID, false)
		require.Equal(t, false, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when allocation flag missing but allocation path is given", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		fileKeyNew := "fileForAllocPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		allocPath := filepath.Join(currentDir, "allocPathForTestS3.txt")
		err = os.WriteFile(allocPath, []byte(allocationID), 0644) //nolint:gosec
		require.Nil(t, err, "allocation file is not written properly")

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"alloc-path": allocPath,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when access key and secret key is missing but aws-cred-path path is given", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})
		fileKeyNew := "fileForAwsCredPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		awsCredPath := filepath.Join(currentDir, "awsCredPathForTestS3.txt")
		lines := []string{
			fmt.Sprintf(`aws_access_key: "%v"`, shared.ConfigData.S3AccessKey),
			fmt.Sprintf(`aws_secret_key: "%v"`, shared.ConfigData.S3SecretKey),
		}
		file, err := os.OpenFile(awsCredPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		require.Nil(t, err, "file is not created properly")
		defer file.Close()

		for _, line := range lines {
			_, err := fmt.Fprintln(file, line)
			require.Nil(t, err, "failed to write file")
		}

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"bucket":        shared.ConfigData.S3BucketNameAlternate,
			"allocation":    allocationID,
			"wallet":        cli_utils.EscapedTestName(t) + "_wallet.json",
			"aws-cred-path": awsCredPath,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass concurrency flag is set to 20", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		fileKeyNew := "fileForConCurrTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key":  shared.ConfigData.S3AccessKey,
			"secret-key":  shared.ConfigData.S3SecretKey,
			"bucket":      shared.ConfigData.S3BucketNameAlternate,
			"wallet":      cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation":  allocationID,
			"concurrency": 20,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass retry flag is set to 4", func(t *test.SystemTest) {
		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		fileKeyNew := "fileForRetryTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"retry":      4,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := shared.CheckStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when filename size is more than 100 character with renamed file names", func(t *test.SystemTest) {
		// Cleanup before test
		err := shared.CleanupBucket(shared.S3Client, shared.ConfigData.S3BucketNameAlternate)
		if err != nil {
			t.Log("Failed to cleanup bucket: ", err)
		}

		allocationID := cli_utils.SetupAllocation(t, shared.ConfigDir, shared.RootPath, map[string]interface{}{
			"size": shared.AllocSize,
		})

		// As per the current logic in s3-migration even the longer file names would be migrated with file names
		// trimmed to 100 chars.
		fileKeyNew := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbb.txt"
		fileContents := []byte("Hello, World!")
		_, err = shared.S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(shared.ConfigData.S3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := cli_utils.MigrateFromCloud(t, cli_utils.CreateParams(map[string]interface{}{
			"access-key": shared.ConfigData.S3AccessKey,
			"secret-key": shared.ConfigData.S3SecretKey,
			"bucket":     shared.ConfigData.S3BucketNameAlternate,
			"wallet":     cli_utils.EscapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))
		// mssg can be changed
		require.Nil(t, err, "Unexpected error", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, shared.ConfigData.S3BucketNameAlternate)
		shortFileKey := shared.GetUniqueShortObjKey(fileKeyNew)
		remoteFilePath = path.Join(remoteFilePath, shortFileKey)
		uploadStats := shared.CheckStats(t, remoteFilePath, shortFileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated does not match with expected file")
	})
}
