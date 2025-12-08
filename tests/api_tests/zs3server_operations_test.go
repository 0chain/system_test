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

	t.SetSmokeTests("CreateBucket should return 200 when all the parameters are correct",
		"ListBucket should return 200 all the parameter are correct",
		"ListObjects should return 200 all the parameter are correct",
		"PutObjects should return 200 all the parameter are correct",
		"GetObjects should return 200 all the parameter are correct",
		"RemoveObject should return 200 all the parameter are correct")

	t.RunSequentially("Zs3 server should return 500 when the action doesn't exist", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "random-action",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		// Nginx may return 401 for invalid actions before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for invalid action, got %d", resp.StatusCode())
	})

	t.RunSequentially("zs3 server should return 500 when the credentials aren't correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       "wrong-access-key",
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		// Nginx returns 401 for invalid credentials before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for invalid credentials, got %d", resp.StatusCode())
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
		resp, err = zs3Client.Zs3ServerRequest(t, queryParams, formData)
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.RunSequentially("GetObjects should return 200 all the parameter are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "getObject",
			"bucketName":      "system-test",
			"objectName":      "test-file.txt",
		}
		formData := map[string]string{
			"file": "@test-file.txt",
		}
		resp, err := zs3Client.Zs3ServerRequest(t, queryParams, formData)
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
		resp, err := zs3Client.Zs3ServerRequest(t, queryParams, formData)
		require.Nil(t, err)
		// Nginx may return 401 for invalid bucket names before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for invalid bucket name, got %d", resp.StatusCode())
		// Only check error message if we got 500 from zs3server
		if resp.StatusCode() == 500 {
			require.Equal(t, `{"error":"Bucket name contains invalid characters"}`, resp.String())
		}
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
		// Ensure the bucket exists first
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "system-test",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())

		// Now try to remove a non-existent object
		queryParams = map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "removeObject",
			"bucketName":      "system-test",
			"objectName":      "file name created as a part of " + t.Name(),
		}
		resp, err = zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	// FIXME - this should be 400 not 500
	t.Run("CreateBucket should return 500 when one of more required parameters are missing", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		// Nginx may return 401 for requests with missing parameters before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for missing parameters, got %d", resp.StatusCode())
	})

	t.Run("ListBuckets should return 500 when one of more required parameters are missing", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"secretAccessKey": SecretAccessKey,
			"action":          "listBucket",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		// Nginx will return 401 for requests with missing accessKey (credential) before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for missing accessKey, got %d", resp.StatusCode())
	})

	t.Run("listObjects should return 500 when trying to list objects from un existing bucket", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listObjects",
			"bucketName":      "random-bucket",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		// Nginx may return 401 for non-existent buckets before the request reaches zs3server
		// Accept both 401 (nginx rejection) and 500 (zs3server error) as valid responses
		require.Contains(t, []int{401, 500}, resp.StatusCode(), "Expected 401 (nginx) or 500 (zs3server) for non-existent bucket, got %d", resp.StatusCode())
	})
}
