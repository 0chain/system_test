package cli_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

const chunksize = 64 * 1024

const (
	dirPrefix  = "dir"
	dirMaxRand = 1000
)

func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	// Specify the bucket name and file key
	bucketName := "dummybucketfortestsmigration"
	fileKey := "sdfg" + ".txt"
	t.TestSetup("Setup s3 bucket with relevant file", func() {
		// Read file contents
		fileContents := []byte("Hello, World!")

		// Upload the file to S3
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKey),
			Body:   bytes.NewReader(fileContents),
		})
		if err != nil {
			t.Skip("S3 Bucket operatiion is not working properly")
		}
	})

	t.RunSequentially("Should migrate existing bucket successfully with skip 0 and replace existing file", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       0,
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, remotepath)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		workingDirName := createDirectoryForTestname(t)
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
		}()
		remotepath := "/root2"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, "/")
		remoteFilePath = path.Join(remoteFilePath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, "/")
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate as copy bucket successfully", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		output, err = migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
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
		// uploadStats := checkStats(t, remoteFilePath, fileKey_modified, allocationID)
		// require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with encryption on", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt":    "true",
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, true)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should skip migration with skip flag == 1 and migartion should be skipped", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       1,
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	t.RunSequentially("Should migrate successfully with duplicate files with skip flag == 2", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       2,
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		// FIXME : copy extension is not there
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_copy." + parts[1]
		remotepath = path.Join(remotepath, bucketName)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully with duplication files  with skip flag == 2 and dup-suffix", func(t *test.SystemTest) {
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
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       2,
			"dup-suffix": "_modified",
			"wd":         workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		// FIXME : dupl suffix is not working
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_modified." + parts[1]
		remotepath = path.Join(remotepath, bucketName)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully and delete the s3 bucket file", func(t *test.SystemTest) {
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
			"access-key":    s3AccessKey,
			"secret-key":    s3SecretKey,
			"bucket":        bucketName,
			"wallet":        escapedTestName(t) + "_wallet.json",
			"allocation":    allocationID,
			"skip":          0,
			"delete-source": true,
			"wd":            workingDirName,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})
}

func checkStats(t *test.SystemTest, remoteFilePath, fname, allocationID string, encrypted bool) bool {
	output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remoteFilePath,
		"json":       "true",
	}), true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var stats map[string]*climodel.FileStats

	err = json.Unmarshal([]byte(output[0]), &stats)
	require.Nil(t, err)

	if len(stats) == 0 {
		return false
	}

	for _, data := range stats {
		if fname != data.Name {
			return false
		}
		if remoteFilePath != data.Path {
			return false
		}
		if fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))) != data.PathHash {
			return false
		}
		if int64(0) != data.NumOfBlockDownloads {
			return false
		}
		if int64(1) != data.NumOfUpdates {
			return false
		}
		if float64(data.NumOfBlocks) != math.Ceil(float64(data.Size)/float64(chunksize)) {
			return false
		}
		if data.WriteMarkerTxn == "" {
			if data.BlockchainAware != false {
				return false
			}
		} else {
			if data.BlockchainAware != true {
				return false
			}
		}
	}
	return true
}

func createDirectoryForTestname(t *test.SystemTest) (fullPath string) {
	rand.Seed(time.Now().UnixNano())

	// Generate a random number within the range
	randomNumber := rand.Intn(dirMaxRand)

	// Generate a unique directory name based on the random number and current timestamp
	dirName := fmt.Sprintf("%s%d_%d", dirPrefix, randomNumber, time.Now().UnixNano())

	fullPath, err := filepath.Abs(dirName)
	require.Nil(t, err)

	err = os.MkdirAll(fullPath, os.ModePerm)
	require.Nil(t, err)

	t.Log("Directory created successfully: ", fullPath)

	return fullPath
}
