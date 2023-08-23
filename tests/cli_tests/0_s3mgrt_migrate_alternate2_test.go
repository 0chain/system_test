package cli_tests

import (
	"bytes"
	"crypto/sha1" // #nosec
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
)

func Test0S3MigrationAlternatePart2(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	fileKey := "OneMinNew" + ".txt"
	t.TestSetup("Setup s3 bucket with relevant file", func() {
		// Cleanup before test
		err := cleanupBucket(S3Client, s3BucketNameAlternate)
		if err != nil {
			t.Log("Failed to cleanup bucket: ", err)
		}
		// Read file contents
		fileContents := []byte("Hello, World!")

		// Upload the file to S3
		_, err = S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKey),
			Body:   bytes.NewReader(fileContents),
		})
		if err != nil {
			t.Skip("S3 Bucket operatiion is not working properly")
		}
	})

	t.RunSequentially("Should migrate existing bucket successfully with concurrency==4 and working dir current dir", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		workingDirName := createDirectoryForTestname(t)
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
		}()
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key":  s3AccessKey,
			"secret-key":  s3SecretKey,
			"bucket":      s3BucketNameAlternate,
			"wallet":      escapedTestName(t) + "_wallet.json",
			"allocation":  allocationID,
			"concurrency": 4,
			"wd":          workingDirName,
		}))

		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with newer than prefix", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "oneMinOld" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"newer-than": time.Now().Unix() - 60, // start timestamp
		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with older than prefix", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		olderThanFileKey := "olderThanFile" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(olderThanFileKey),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		cliutils.Wait(t, 70*time.Second)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"older-than": time.Now().Unix() - 60, // end timestamp
		}))

		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, olderThanFileKey)
		uploadStats := checkStats(t, remoteFilePath, olderThanFileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with files staring with given prefix", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyToBemigrated := "mgrt" + ".txt"
		fileKeyNotToBeMigrated := "noMgrt" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyToBemigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)
		_, err = S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNotToBeMigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"prefix":     "mgrt",
		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePathPos := path.Join(remoteFilePath, fileKeyToBemigrated)
		remoteFilePathNeg := path.Join(remoteFilePath, fileKeyNotToBeMigrated)
		uploadStats := checkStats(t, remoteFilePathPos, fileKeyToBemigrated, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
		uploadStats = checkStats(t, remoteFilePathNeg, fileKeyNotToBeMigrated, allocationID, false)
		require.Equal(t, false, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when allocation flag missing but allocation path is given", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForAllocPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		allocPath := filepath.Join(currentDir, "allocPathForTestS3.txt")
		err = os.WriteFile(allocPath, []byte(allocationID), 0644) //nolint:gosec
		require.Nil(t, err, "allocation file is not written properly")

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"alloc-path": allocPath,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when access key and secret key is missing but aws-cred-path path is given", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		fileKeyNew := "fileForAwsCredPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		awsCredPath := filepath.Join(currentDir, "awsCredPathForTestS3.txt")
		lines := []string{
			fmt.Sprintf(`aws_access_key: "%v"`, s3AccessKey),
			fmt.Sprintf(`aws_secret_key: "%v"`, s3SecretKey),
		}
		file, err := os.OpenFile(awsCredPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		require.Nil(t, err, "file is not created properly")
		defer file.Close()

		for _, line := range lines {
			_, err := fmt.Fprintln(file, line)
			require.Nil(t, err, "failed to write file")
		}

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"bucket":        s3BucketNameAlternate,
			"allocation":    allocationID,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"aws-cred-path": awsCredPath,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass concurrency flag is set to 20", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForConCurrTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key":  s3AccessKey,
			"secret-key":  s3SecretKey,
			"bucket":      s3BucketNameAlternate,
			"wallet":      escapedTestName(t) + "_wallet.json",
			"allocation":  allocationID,
			"concurrency": 20,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass retry flag is set to 4", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForRetryTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"retry":      4,
		}))

		require.Nil(t, err, "Expected a Migration completed successfully but got error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Migration completed successfully", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when filename size is more than 100 character with renamed file names", func(t *test.SystemTest) {
		// Cleanup before test
		err := cleanupBucket(S3Client, s3BucketNameAlternate)
		if err != nil {
			t.Log("Failed to cleanup bucket: ", err)
		}

		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		// As per the current logic in s3-migration even the longer file names would be migrated with file names
		// trimmed to 100 chars.
		fileKeyNew := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbb.txt"
		fileContents := []byte("Hello, World!")
		_, err = S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))
		// mssg can be changed
		require.Nil(t, err, "Unexpected error", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		shortFileKey := getUniqueShortObjKey(fileKeyNew)
		remoteFilePath = path.Join(remoteFilePath, shortFileKey)
		uploadStats := checkStats(t, remoteFilePath, shortFileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated does not match with expected file")
	})
}

// This func is a copy of new file name generation logic in s3-migration from migration/migrate.go
// This func needs to be in sync with the original func.
func getUniqueShortObjKey(objectKey string) string {
	// Max length to which objectKey would be trimmed to.
	const maxLength = 100

	if len(objectKey) > maxLength {
		// Generate a SHA-1 hash of the object key
		hash := sha1.New() // #nosec
		_, _ = hash.Write([]byte(objectKey))
		hashSum := hash.Sum(nil)

		// Convert the hash to a hexadecimal string
		hashString := hex.EncodeToString(hashSum)

		// Combine the first 10 characters of the hash with a truncated object key
		shortKey := fmt.Sprintf("%s_%s", hashString[:10], objectKey[11+len(objectKey)-maxLength:])
		return shortKey
	}

	return objectKey
}
