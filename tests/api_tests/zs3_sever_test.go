package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestCreateBucket(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("run createbucket test", func(t *test.SystemTest) {
		resp, err := zs3Client.CreateBucket(t)
		require.Nil(t, err)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
	})
}

func TestListBuckets(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("run ListBuckets test", func(t *test.SystemTest) {
		resp, err := zs3Client.ListBucket(t)
		require.Nil(t, err)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
	})
}

func TestPutObject(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("run ListBuckets test", func(t *test.SystemTest) {
		resp, err := zs3Client.PutObject(t)
		require.Nil(t, err)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)

		resp, err = zs3Client.RemoveObject(t)
		require.Nil(t, err)
		require.Equal(t, resp.StatusCode(), client.HttpOkStatus)
	})
}
