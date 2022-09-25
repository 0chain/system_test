package api_tests

import (
	"github.com/stretchr/testify/require"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestAtlusChimneyGraphTotalChallengePools(t *testing.T) {
	t.Parallel()

	t.Run("test api should return error when event db not able to find round matching", func(t *testing.T) {
		t.Parallel()
		from := ""
		to := ""

		resp, httpResp, httpErr := v1SCRestGetDataPointsUsingURL(t, createParams(map[string]interface{}{
			"url":         "/graph-total-challenge-pools",
			"data-points": "1",
			"from":        from,
			"to":          to,
		}))

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing data points param", func(t *testing.T) {
		t.Parallel()
		from := ""
		to := ""

		resp, httpResp, httpErr := v1SCRestGetDataPointsUsingURL(t, createParams(map[string]interface{}{
			"url":  "/graph-total-challenge-pools",
			"from": from,
			"to":   to,
		}))

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing from param", func(t *testing.T) {
		t.Parallel()
		to := ""

		resp, httpResp, httpErr := v1SCRestGetDataPointsUsingURL(t, createParams(map[string]interface{}{
			"url":         "/graph-total-challenge-pools",
			"data-points": "1",
			"to":          to,
		}))

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing to param", func(t *testing.T) {
		t.Parallel()
		from := ""

		resp, httpResp, httpErr := v1SCRestGetDataPointsUsingURL(t, createParams(map[string]interface{}{
			"url":         "/graph-total-challenge-pools",
			"data-points": "1",
			"from":        from,
		}))

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should run successfully", func(t *testing.T) {
		t.Parallel()
		from := strconv.FormatInt(int64(math.Floor(float64(time.Now().Unix()-86400000/1000))), 10)
		to := strconv.FormatInt(int64(math.Floor(float64(time.Now().Unix()/1000))), 10)

		resp, httpResp, httpErr := v1SCRestGetDataPointsUsingURL(t, createParams(map[string]interface{}{
			"url":         "/graph-total-challenge-pools",
			"data-points": "1",
			"from":        from,
			"to":          to,
		}))

		require.Nil(t, httpErr)
		require.Equal(t, http.StatusOK, httpResp.StatusCode())
		require.NotNil(t, resp)
	})
}
