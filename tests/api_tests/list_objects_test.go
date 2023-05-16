package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestListObjects(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("ListObjects should return 200 when all the parameters are correct")
	t.Parallel()
	t.Run("ListObjects should return 200 when all the parameters are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listObjects",
			"bucketName":      "root",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
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
		require.Equal(t, 500, resp.StatusCode())
	})
}
