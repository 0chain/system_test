package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
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
func Test0S3MigrationAlternate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if s3SecretKey == "" || s3AccessKey == "" {
		t.Skip("s3SecretKey or s3AccessKey was missing")
	}
	t.Parallel()
	t.SetSmokeTests("Should migrate existing bucket successfully")

	t.Run("Should migrate as copy bucket successfully ", func(t *test.SystemTest) {
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

		output, err = migrateFromS3(t, configPath, createParams(map[string]interface{}{
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
	})

	t.Run("Should migrate existing bucket to specified path successfully with encryption on", func(t *test.SystemTest) {
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
			"migarate-to": "/root2",
			"encrypt": "true",
		}))

		require.Nil(t, err, "Unexpected migration failure", strings.Join(output, "\n"))
		require.Equal(t, len(output), 1, "More/Less output was returned than expected", strings.Join(output, "\n"))
		require.Contains(t, "Migration completed successfully", output[0], "Output was not as expected", strings.Join(output, "\n"))
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
		filesize := int64(555)
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)
		fname := filepath.Base(filename)
		remoteFilePath := path.Join(remotepath, fname)

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
			require.Equal(t, fname, data.Name)
			require.Equal(t, remoteFilePath, data.Path)
			require.Equal(t, fmt.Sprintf("%x", sha3.Sum256([]byte(allocationID+":"+remoteFilePath))), data.PathHash)
			require.Equal(t, int64(0), data.NumOfBlockDownloads)
			require.Equal(t, int64(1), data.NumOfUpdates)
			require.Equal(t, float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(chunksize)))
			if data.WriteMarkerTxn == "" {
				require.Equal(t, false, data.BlockchainAware)
			} else {
				require.Equal(t, true, data.BlockchainAware)
			}

			// FIXME: POSSIBLE BUG: key name and blobberID in value should be same but this is not consistent for every run and happening randomly
			// require.Equal(t, blobberID, data.BlobberID, "key name and blobberID in value should be same")
		}
	})


}
