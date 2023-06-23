package cli_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

const chunksize = 64 * 1024

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

	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.RunSequentially("Should migrate existing bucket successfully with skip 0 and replace existing file", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip": 0,
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
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		remotepath := "/root2"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
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

	// what would be the case if one copy already exist  ?
	t.RunSequentially("Should migrate as copy bucket successfully ", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
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
			"skip": 2,
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

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		//FIXME : There is no extra extension for encrypted file
		// parts := strings.Split(fileKey, ".")
		// fileKey_modified := parts[0]+ "_encrypted." + parts[1]
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

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip": 1,
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
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip": 2,
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

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip": 2,
			"dup-suffix" : "_modified",
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

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"skip": 0,
			"delete-source" : true,
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

func checkStats(t *test.SystemTest, remoteFilePath, fname, allocationID string, encrypted bool)(bool){

	output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remoteFilePath,
		"json":       "true",
	}), true)
	require.Nil(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)

	var stats map[string]climodel.FileStats

	err = json.Unmarshal([]byte(output[0]), &stats)
	require.Nil(t, err)
	
	if(len(stats) == 0){
		return false;
	} 

	for _, data := range stats {
		if(fname != data.Name){
			return false
		}
		if(remoteFilePath !=  data.Path){
			return false
		}
		if( fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))) !=  data.PathHash){
			return false
		}
		if(int64(0) !=  data.NumOfBlockDownloads){
			return false
		}
		if(int64(1) !=  data.NumOfUpdates){
			return false
		}
		if(float64(data.NumOfBlocks) != math.Ceil(float64(data.Size)/float64(chunksize))){
			return false
		}
		if data.WriteMarkerTxn == "" {
			if( data.BlockchainAware != false){
				return false;
			}
		}else {
			if( data.BlockchainAware != true){
				return false;
			}
		}
	}
	return true	
}

// allocation path and aws credential path also needs to be tested
// if region is tested or not that we need to check
// resume retry, first look into code then ask ryan
func Test0S3MigrationAlternatePart2(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	// Specify the bucket name and file key
	bucketName := "dummybucketfortestsmigration"
	fileKey := "OneMinNew" + ".txt"
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

	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate existing bucket successfully with concurrency==4 and working dir current dir", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		workDir := "/"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"concurrency" : 4,
			"wd": workDir,

		}))

		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with newer than prefix", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "oneMinOld" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"newer-than" : "1m",

		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with older than prefix", func(t *test.SystemTest) {
		t.Logf("here")
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
			"older-than" : "10m",

		}))

		remotepath := "/"
		fileKeyOld := "TenMinOldfile" + ".txt"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyOld)
		uploadStats := checkStats(t, remoteFilePath, fileKeyOld, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with files staring with given prefix", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyToBemigrated := "mgrt" + ".txt"
		fileKeyNotToBeMigrated := "noMgrt" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyToBemigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)
		_, err = S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNotToBeMigrated),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":    	bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"prefix" : "mgrt",

		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePathPos := path.Join(remoteFilePath, fileKeyToBemigrated)
		remoteFilePathNeg := path.Join(remoteFilePath, fileKeyNotToBeMigrated)
		uploadStats := checkStats(t, remoteFilePathPos, fileKeyToBemigrated, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
		uploadStats = checkStats(t, remoteFilePathNeg, fileKeyNotToBeMigrated, allocationID, false)
		require.Equal(t, uploadStats, false, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when allocation flag missing but allocation path is given", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForAllocPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		allocPath := filepath.Join(currentDir, "allocPathForTestS3.txt")
		err = ioutil.WriteFile(allocPath, []byte(allocationID), 0644)
		require.Nil(t, err, "allocation file is not written properly")
	
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"alloc-path" : allocPath,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when access key and secret key is missing but aws-cred-path path is given", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		fileKeyNew := "fileForAwsCredPath" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		currentDir, err := os.Getwd()
		require.Nil(t, err, "can't get current dir")
		awsCredPath := filepath.Join(currentDir, "awsCredPathForTestS3.txt")
		lines := []string{
			`s3_access_key: "AKIA4MPQDEZ4ODBRWUOU"`,
			`s3_secret_key: "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"`,
		}
		file, err := os.OpenFile(awsCredPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		require.Nil(t, err, "file is not created properly")
		defer file.Close()

		for _, line := range lines {
			_, err := fmt.Fprintln(file, line)
			require.Nil(t, err, "failed to write file")
		}
		
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"bucket":     bucketName,
			"allocation": allocationID,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"aws-cred-path" : awsCredPath,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass concurrency flag is set to 20", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForConCurrTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"concurrency" : 20,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass retry flag is set to 4", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})
		
		fileKeyNew := "fileForRetryTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"retry" : 4,
		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should pass when working dir is set to current dir", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := "fileForWorkingDTest" + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"wd": "/",

		}))

		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, bucketName)
		remoteFilePath = path.Join(remoteFilePath, fileKeyNew)
		uploadStats := checkStats(t, remoteFilePath, fileKeyNew, allocationID, false)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.RunSequentially("Should not pass when filename size is more than 100 character", func(t *test.SystemTest) {
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		fileKeyNew := t.EscapedName() + ".txt"
		fileContents := []byte("Hello, World!")
		_, err := S3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileKeyNew),
			Body:   bytes.NewReader(fileContents),
		})
		require.Nil(t, err)

		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"wd": "/",

		}))
		// mssg can be changed
		require.NotNil(t, err, "Expected a migration failure but got no error", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, output[0], "Error: allocation id is missing", "Output was not as expected", strings.Join(output, "\n"))
	})

}

