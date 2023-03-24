package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestZs3ServerOpertions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	//t.Parallel()
	t.Run("CreateBucket should return 200 when all the parameters are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "system-test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.Run("ListBucket should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listBuckets",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.Run("ListObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listObjects",
			"bucketName":      "system-test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
	t.Run("PutObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "putObject",
			"bucketName":      "system-test",
		}
		formData := map[string]string{
			"file": "@test-file",
		}
		resp, err := zs3Client.PutObject(t, queryParams, formData)
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
	// t.Run("GetObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
	// 	queryParams := map[string]string{
	// 		"accessKey":       AccessKey,
	// 		"secretAccessKey": SecretAccessKey,
	// 		"action":          "getObject",
	// 		"bucketName":      "system-test",
	// 		"objectName":      "test-file",
	// 	}
	// 	resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
	// 	require.Nil(t, err)
	// 	require.Equal(t, 200, resp.StatusCode())
	// })
	t.Run("RemoveObject should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "removeObject",
			"bucketName":      "system-test",
			"objectName":      "test-file",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
}
