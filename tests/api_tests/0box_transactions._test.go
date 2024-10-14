package api_tests

import (
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxTransactions(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("get paginated transactions list while creating pit id", 5*time.Minute, func(t *test.SystemTest) {
		txnData, resp, err := zboxClient.GetTransactionsList(t, "")
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode())

		if txnData != nil {
			pitId := txnData.PitId
			_, resp, err := zboxClient.GetTransactionsList(t, pitId)
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
		}
	})
}
