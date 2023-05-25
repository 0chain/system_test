package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

const (
	AccessKey       = "rootroot"
	SecretAccessKey = "rootroot"
)

func TestZs3ServerOperations(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()
	// FIXME: we should never return a 500 to the end user
	t.Run("Zs3 server should return 500 when the action doesn't exist", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "random-action",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 500, resp.StatusCode())
	})

	t.Run("zs3 server should return 500 when the credentials aren't correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       "wrong-access-key",
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 500, resp.StatusCode())
	})

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

	t.RunSequentially("CreateBucket should not return error when bucket name already exist", func(t *test.SystemTest) {
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

	t.RunSequentially("PutObjects should return error when buckcet name does not exist", func(t *test.SystemTest) {
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
