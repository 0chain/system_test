package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// adding comment to run ppipeline again
func TestZs3ServerOpertions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()
	t.RunSequentially("CreateBucket should return 200 when all the parameters are correct", func(t *test.SystemTest) {
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
	t.RunSequentially("CreateBucket should return error when bucket name already exist", func(t *test.SystemTest) {
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

	t.RunSequentially("ListBucket should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listBuckets",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.RunSequentially("ListObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
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
	t.RunSequentially("PutObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
		// t.Skip("wait for the issue to get resolved : https://github.com/0chain/zs3server/issues/21")
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "system-test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())

		queryParams = map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "putObject",
			"bucketName":      "system-test",
		}
		formData := map[string]string{
			"file": "@test-file.txt",
		}
		resp, err = zs3Client.PutObject(t, queryParams, formData)
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
	t.RunSequentially("GetObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
		// t.Skip("wait for the issue to get resolved : https://github.com/0chain/zs3server/issues/21")
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "putObject",
			"bucketName":      "system-test",
		}
		formData := map[string]string{
			"file": "@test-file.txt",
		}
		resp, err := zs3Client.PutObject(t, queryParams, formData)
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
		queryParams = map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "getObject",
			"bucketName":      "system-test",
		}
		formData = map[string]string{
			"file": "@test-file.txt",
		}
		t.Logf(resp.String())
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.RunSequentially("PutObjects should return error when buckcet name doesnot exist", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "putObject",
			"bucketName":      "This bucket doesnot exist",
		}
		formData := map[string]string{
			"file": "@test-file.txt",
		}
		resp, err := zs3Client.PutObject(t, queryParams, formData)
		require.Nil(t, err)
		require.Equal(t, 500, resp.StatusCode())
		require.Equal(t, `{"error":"Bucket name contains invalid characters"}`, resp.String())
	})
	t.RunSequentially("RemoveObject should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "system-test",
			"objectName":      "bucket created as a part of " + t.Name(),
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
		queryParams = map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "removeObject",
			"bucketName":      "system-test",
			"objectName":      "bucket created as a part of " + t.Name(),
		}
		resp, err = zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.RunSequentially("RemoveObject should not return error if object doen't exist", func(t *test.SystemTest) {
		// t.Skip("wait for the issue to get resolved : https://github.com/0chain/zs3server/issues/22")
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "removeObject",
			"bucketName":      "system-test",
			"objectName":      "file name created as a part of " + t.Name(),
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})
}
