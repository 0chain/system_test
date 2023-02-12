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
		require.Equal(t, client.HttpOkStatus, resp.StatusCode())
	})
}

func TestListBuckets(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("run ListBuckets test", func(t *test.SystemTest) {
		resp, err := zs3Client.ListBucket(t)
		require.Nil(t, err)
		require.Equal(t, client.HttpOkStatus, resp.StatusCode())
	})
}

func TestPutObject(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()
	t.Run("putObject should upload file succefully when we pass the correct param", func(t *test.SystemTest) {
		resp, err := zs3Client.PutObject(t)
		require.Nil(t, err)
		require.Equal(t, client.HttpOkStatus, resp.StatusCode())

		resp, err = zs3Client.RemoveObject(t)
		require.Nil(t, err)
		require.Equal(t, client.HttpOkStatus, resp.StatusCode())
	})
}
