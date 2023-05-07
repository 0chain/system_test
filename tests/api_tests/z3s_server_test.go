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

func TestZs3Server(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Zs3 server should return 500 when the action doesn't exist")

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
}
