package shared

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
)

func CleanupBucket(svc *s3.S3, s3BucketNameAlternate string) error {
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

func CheckStats(t *test.SystemTest, remoteFilePath, fname, allocationID string, encrypted bool) bool {
	t.Log("remotepath: ", remoteFilePath)
	output, err := cli_utils.GetFileStats(t, ConfigPath, cli_utils.CreateParams(map[string]interface{}{
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
		if float64(data.NumOfBlocks) != math.Ceil(float64(data.Size)/float64(Chunksize)) {
			t.Logf("6. %f != %f", float64(data.NumOfBlocks), math.Ceil(float64(data.Size)/float64(Chunksize)))
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

func CreateDirectoryForTestname(t *test.SystemTest) (fullPath string) {
	randomBigInt, err := rand.Int(rand.Reader, big.NewInt(int64(DirMaxRand)))
	require.Nil(t, err)

	randomNumber := int(randomBigInt.Int64())

	// Generate a unique directory name based on the random number and current timestamp
	dirName := fmt.Sprintf("%s%d_%d", DirPrefix, randomNumber, time.Now().UnixNano())

	fullPath, err = filepath.Abs(dirName)
	require.Nil(t, err)

	err = os.MkdirAll(fullPath, os.ModePerm)
	require.Nil(t, err)

	t.Log("Directory created successfully: ", fullPath)

	return fullPath
}

func GetUniqueShortObjKey(objectKey string) string {
	// Max length to which objectKey would be trimmed to.
	// Keeping this less than 100 chars to prevent longer name in case of uploading duplicate
	// files with `_copy` suffixes.
	const maxLength = 90

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
