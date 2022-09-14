package api_tests

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestName(t *testing.T) {
	t.Parallel()

	t.Run("Get miner stats call should return successfully", func(t *testing.T) {
		t.Parallel()
		stats, httpResponse, err := api_client.v1MinerGetStats(t, api_client.ConsensusByHttpStatus(api_client.HttpOkStatus))

		require.Nil(t, err)
		require.Equal(t, api_client.HttpOkStatus, httpResponse.Status(), httpResponse)
		require.NotNil(t, stats)
		require.Greater(t, stats.BlockFinality, float64(0), httpResponse)
		require.Greater(t, stats.LastFinalizedRound, int64(0), httpResponse)
		require.Greater(t, stats.BlocksFinalized, int64(0), httpResponse)
		require.GreaterOrEqual(t, stats.StateHealth, int64(-1), httpResponse)
		require.Greater(t, stats.CurrentRound, int64(0), httpResponse)
		require.GreaterOrEqual(t, stats.RoundTimeout, int64(0), httpResponse)
		require.GreaterOrEqual(t, stats.Timeouts, int64(0), httpResponse)
		require.Greater(t, stats.AverageBlockSize, 0, httpResponse)
		require.NotNil(t, stats.NetworkTime, httpResponse)
	})

	t.Run("Get sharder stats call should return successfully", func(t *testing.T) {
		t.Parallel()
		stats, httpResponse, err := api_client.v1SharderGetStats(t, api_client.ConsensusByHttpStatus(api_client.HttpOkStatus))

		require.Nil(t, err)
		require.Equal(t, api_client.HttpOkStatus, httpResponse.Status(), httpResponse)
		require.NotNil(t, stats)
		require.Greater(t, stats.LastFinalizedRound, int64(0), httpResponse)
		require.GreaterOrEqual(t, stats.StateHealth, int64(-1), httpResponse)
		require.Greater(t, stats.AverageBlockSize, 0, httpResponse)
		require.GreaterOrEqual(t, stats.PrevInvocationCount, uint64(0), httpResponse)
		require.NotNil(t, stats.PrevInvocationScanTime, uint64(0), httpResponse)
		require.GreaterOrEqual(t, stats.MeanScanBlockStatsTime, float64(0), httpResponse)
	})
}
