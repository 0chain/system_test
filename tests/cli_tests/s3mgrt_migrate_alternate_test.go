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
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

/*
Let's list down the cases
delete source is not covered let's check
--dup_suffix
how to verify --encrypt option I think you need to see the code
--migrate to how to create that remote path
-- how to verify older than, just create older than 10 min file
-- resume  what do you mean by previous state
-- retry count , how to verify it, don't you think that retry would be one in ideal case so how to increase the retry count for testing purpose
-- skip can be tested
-- but what is duplicate
-- what is this wd
*/
//concurrency cann't be tested
//dellete source can be tested
// skip can be tested and there would be three cases for this case0, 1, 2, but how to check duplicate file
//migrate to can be tested also if somehow skip is tested
// encrption need to figure out
// --dup-suffix this can also be tested
// newer than , prefix and older than can also be tested
const chunksize = 64 * 1024

func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}

	// Specify the bucket name and file key
	bucketName := "dummybucketfortestsmigration"
	fileKey := t.Name() + ".txt"

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

	//t.Parallel()
	// t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
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
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		fileKey = fileKey + "encrypt"
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

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
		fileKey = fileKey + "_copy"
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
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
		fileKey = fileKey + "_encrypt"
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

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

	t.Run("Should skip migration with skip flag == 1", func(t *test.SystemTest) {
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
		fileKey = fileKey + "_copy"
		// add dupl prefix
		remoteFilePath := path.Join(remotepath, fileKey)
		uploadStats := checkStats(t, remoteFilePath, fileKey, allocationID)
		require.Equal(t, uploadStats, true, "The file migrated doesnot match with with required file")
	})

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
			"skip": 2,
			"delete-source" : true,
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))

		remotepath := "/"
		fileKey = fileKey + "_copy"
		// add dupl prefix
		remoteFilePath := path.Join(remotepath, fileKey)
		
		fileStat, err := getFileStats(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remoteFilePath,
			"json":       "",
		}), true)
		require.Nil(t, err, strings.Join(fileStat, "\n"))
		require.Len(t, output, 1)
	
		var stats map[string]climodel.FileStats
	
		err = json.Unmarshal([]byte(fileStat[0]), &stats)
		require.Nil(t, err)
		require.Equal(t, len(stats), 0)
		
		
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