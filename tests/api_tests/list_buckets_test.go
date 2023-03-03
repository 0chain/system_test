package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestListBuckets(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Parallel()
	t.Run("ListBuckets should return 200 when all the parameters are correct", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"accessKey":       AccessKey,
			"secretAccessKey": SecretAccessKey,
			"action":          "listBuckets",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 200, resp.StatusCode())
	})

	t.Run("ListBuckets should return 500 when one of more required parameters are missing", func(t *test.SystemTest) {
		queryParams := map[string]string{
			"secretAccessKey": SecretAccessKey,
			"action":          "listBucket",
		}
		resp, err := zs3Client.BucketOperation(t, queryParams, map[string]string{})
		require.Nil(t, err)
		require.Equal(t, 500, resp.StatusCode())
	})
}
