package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestCreateBucket(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("CreateBucket should return 200 when all the parameters are correct")
	t.Parallel()
	t.Run("CreateBucket should return 200 when all the parameters are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "createBucket",
			"bucketName":      "system-test1",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
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
		require.Equal(t, 500, resp.StatusCode())
	})
}
