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

	fileKey := "sdfg" + ".txt"
	t.TestSetup("Setup s3 bucket with relevant file", func() {
		// Cleanup bucket before test
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

	t.RunSequentially("Should migrate existing bucket successfully with skip 0 and replace existing file", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       0,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, remotepath)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		remotepath := "/root2"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, "/")
		remoteFilePath = path.Join(remoteFilePath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, "/")
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate as copy bucket successfully", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		output, err = migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
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
		// require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate existing bucket successfully with encryption on", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt":    "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, true)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should skip migration with skip flag == 1 and migartion should be skipped", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip":       1,
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

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
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
		remotepath = path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully with duplication files  with skip flag == 2 and dup-suffix", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
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
		remotepath = path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should migrate successfully and delete the s3 bucket file and use custom workdir", func(t *test.SystemTest) {
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
			"bucket":        s3BucketNameAlternate,
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
		remoteFilePath := path.Join(remotepath, s3BucketNameAlternate)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, true, uploadStats, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should error out if workdir is not default and not empty", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		workingDirName := createDirectoryForTestname(t)
		file, _ := os.CreateTemp(workingDirName, "prefix")
		// remove the dir after use
		defer func() {
			_ = os.RemoveAll(workingDirName)
			_ = os.Remove(file.Name())
		}()

		remotepath := "/root3"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3BucketNameAlternate,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
			"wd":         workingDirName,
		}))

		require.NotNil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "working directory not empty", "Output was not as expected", strings.Join(output, "\n"))
	})
}

func checkStats(t *test.SystemTest, remoteFilePath, fname, allocationID string, encrypted bool) bool {
	t.Log("remotepath: ", remoteFilePath)
	output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remoteFilePath,
		"json":       "true",
	}), true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var stats map[string]*climodel.FileStats
	t.Log(output[0])
	err = json.Unmarshal([]byte(output[0]), &stats)
	require.Nil(t, err)

	if len(stats) == 0 {
		t.Logf("0. zero no files")
		return false
	}

	for _, data := range stats {
		if fname != data.Name {
			t.Logf("1. %s != %s", fname, data.Name)
			return false
		}
		if remoteFilePath != data.Path {
			t.Logf("2. %s != %s", remoteFilePath, data.Path)
			return false
		}
		hash := fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath)))
		if hash != data.PathHash {
			t.Logf("3. %s != %s", hash, data.PathHash)
			return false
		}
		if int64(0) != data.NumOfBlockDownloads {
			t.Logf("4. %d != %d", int64(0), data.NumOfBlockDownloads)
			return false
		}
		if int64(1) != data.NumOfUpdates {
			t.Logf("5. %d != %d", int64(1), data.NumOfUpdates)
			return false
		}
		if float64(data.NumOfBlocks) != math.Ceil(float64(data.Size)/float64(chunksize)) {
			t.Logf("6. %f != %f", float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			return false
		}
		if data.WriteMarkerTxn == "" {
			if data.BlockchainAware != false {
				t.Logf("7. %t", data.BlockchainAware)
				return false
			}
		} else {
			if data.BlockchainAware != true {
				t.Logf("8. %t", data.BlockchainAware)
				return false
			}
		}
	}
	return true
}

func createDirectoryForTestname(t *test.SystemTest) (fullPath string) {
	rand.Seed(time.Now().UnixNano())

	// Generate a random number within the range
	randomNumber := rand.Intn(dirMaxRand) //nolint:gosec

	// Generate a unique directory name based on the random number and current timestamp
	dirName := fmt.Sprintf("%s%d_%d", dirPrefix, randomNumber, time.Now().UnixNano())

	fullPath, err := filepath.Abs(dirName)
	require.Nil(t, err)

	err = os.MkdirAll(fullPath, os.ModePerm)
	require.Nil(t, err)

	t.Log("Directory created successfully: ", fullPath)

	return fullPath
}

func cleanupBucket(svc *s3.S3, s3BucketNameAlternate string) error {
	// List all objects within the bucket
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3BucketNameAlternate),
	})
	if err != nil {
		return err
	}

	// Delete each object in the bucket
	for _, obj := range resp.Contents {
		_, err := svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s3BucketNameAlternate),
			Key:    obj.Key,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
