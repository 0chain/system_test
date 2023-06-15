package cli_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/coredns/coredns/plugin/pkg/log"
	"github.com/google/martian/v3/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

/*

-- how to verify older than, just create older than 10 min file
-- resume  what do you mean by previous state
-- retry count , how to verify it, don't you think that retry would be one in ideal case so how to increase the retry count for testing purpose
*/
// i think working dir wd can be tested with concurrency

//delete source done
// skip done
// migrate to done
// encrption done
// --dup-suffix done
// newer than , prefix and older than , concurrency, resume , retry count I will test in next pr

const chunksize = 64 * 1024

func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	// Specify the bucket name and file key
	bucketName := "dummybucketfortestsmigration"
	fileKey := t.Name() + ".txt"
	setupTest(t, bucketName, fileKey)
	
	t.Cleanup(func() {
		input := &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		}
	
		// Delete the bucket
		_, err := S3Client.DeleteBucket(input)
		if err != nil {
			log.Info("Failed to delete bucket:", err)
			return
		}
	
		log.Info("Bucket", bucketName, "deleted successfully!")
	})


	// t.Parallel()
	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		remotepath := "/root2"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"migrate-to": remotepath,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	// what would be the case if one copy already exist  ?
	t.Run("Should migrate as copy bucket successfully ", func(t *test.SystemTest) {
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
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		parts := strings.Split(fileKey, ".")
		fileKey_modified := parts[0]+ "_modified." + parts[1]
		remoteFilePath := path.Join(remotepath, fileKey_modified)
		uploadStats := checkStats(t, remoteFilePath, fileKey_modified, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
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

		remotepath := "/"
		// check if _encrypt is correct extension for encrypted file
		parts := strings.Split(fileKey, ".")
		fileKey_modified := parts[0]+ "_encrypted." + parts[1]
		remoteFilePath := path.Join(remotepath, fileKey_modified)
		uploadStats := checkStats(t, remoteFilePath, fileKey_modified, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	// done
	t.Run("Should migrate existing bucket successfully with skip 0  and replace existing", func(t *test.SystemTest) {
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
			"skip": 0,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	// done
	t.Run("Should skip migration with skip flag == 1 and migartion should be skipped", func(t *test.SystemTest) {
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
			"skip": 1,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		// change the massege please
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
	})

	// done
	t.Run("Should migrate successfully with duplication files  with skip flag == 2", func(t *test.SystemTest) {
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
			"skip": 2,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		parts := strings.Split(fileKey, ".")
		fileKey_modified := parts[0]+ "_copy." + parts[1]
		remoteFilePath := path.Join(remotepath, fileKey_modified)
		uploadStats := checkStats(t, remoteFilePath, fileKey_modified, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	// done
	t.Run("Should migrate successfully with duplication files  with skip flag == 2 and dup-suffix", func(t *test.SystemTest) {
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
			"skip": 2,
			"dup-suffix" : "_modified",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		parts := strings.Split(fileKey, ".")
		fileKey_modified := parts[0]+ "_modified." + parts[1]
		remoteFilePath := path.Join(remotepath, fileKey_modified)
		uploadStats := checkStats(t, remoteFilePath, fileKey_modified, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	// done
	t.Run("Should migrate successfully and delete the s3 bucket file", func(t *test.SystemTest) {
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
			"skip": 0,
			"delete-source" : true,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")

		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		}
	
		// Retrieve the list of objects in the bucket
		result, err := S3Client.ListObjectsV2(input)
		if err != nil {
			fmt.Println("Failed to list objects:", err)
			return
		}
	
		// Iterate over the objects to check if the file exists
		fileExists := false
		for _, obj := range result.Contents {
			if *obj.Key == fileKey {
				fileExists = true
				break
			}
		}
	
		require.Equal(t, fileExists, false, "file is not deleted from source")
		
	})


}

func checkStats(t *test.SystemTest, remoteFilePath, fname, allocationID string)(bool){

	output, err := getFileStats(t, configPath, createParams(map[string]interface{}{
		"allocation": allocationID,
		"remotepath": remoteFilePath,
		"json":       "",
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

func setupTest(t *test.SystemTest, bucketName, fileKey string) {
	// Read file contents
	fileContents := []byte("Hello, World!")

	// Upload the file to S3
	_, err := S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileKey),
		Body:   bytes.NewReader(fileContents),
	})
	if err != nil {
		fmt.Println("Failed to create file in S3:", err)
		t.Skip("S3 Bucket operatiion is not working properly")
	}

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
	fileKey := t.Name() + ".txt"
	setupTest(t, bucketName, fileKey)
	
	t.Cleanup(func() {
		input := &s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		}
	
		// Delete the bucket
		_, err := S3Client.DeleteBucket(input)
		if err != nil {
			log.Info("Failed to delete bucket:", err)
			return
		}
	
		log.Info("Bucket", bucketName, "deleted successfully!")
	})


	// t.Parallel()
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
		// statFilePath := fmt.Sprintf("%v/%v.state", workDir, bucketName)
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with newer than prefix", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		//workDir := "/"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"newer-than" : "10m",

		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with older than prefix", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		//workDir := "/"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"older-than" : "10m",

		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

	t.Run("Should migrate existing bucket successfully with files staring with given prefix", func(t *test.SystemTest) {
		t.Logf("here")
		allocSize := int64(50 * MB)
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		//workDir := "/"
		output, err := migrateFromS3(t, configPath, createParams(map[string]interface{}{
			"access-key": s3AccessKey,
			"secret-key": s3SecretKey,
			"bucket":     s3bucketName,
			"wallet":     escapedTestName(t) + "_wallet.json",
			"allocation": allocationID,
			"prefix" : "hello",

		}))
		remotepath := "/"
		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

}

