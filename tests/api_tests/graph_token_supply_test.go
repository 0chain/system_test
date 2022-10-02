package api_tests

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/stretchr/testify/require"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestGraphTokenSupply(t *testing.T) {
	t.Parallel()

	t.Run("test api should return error when event db not able to find round matching", func(t *testing.T) {
		t.Parallel()
		from := ""
		to := ""

		resp, httpResp, httpErr := apiClient.V1SCRestGetDataPointsUsingURL(createParams(map[string]interface{}{
			"url":         client.GetGraphTokenSupply,
			"data-points": "1",
			"from":        from,
			"to":          to,
		}), http.StatusOK)

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing data points param", func(t *testing.T) {
		t.Parallel()
		from := ""
		to := ""

		resp, httpResp, httpErr := apiClient.V1SCRestGetDataPointsUsingURL(createParams(map[string]interface{}{
			"url":  client.GetGraphTokenSupply,
			"from": from,
			"to":   to,
		}), http.StatusOK)

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing from param", func(t *testing.T) {
		t.Parallel()
		to := ""

		resp, httpResp, httpErr := apiClient.V1SCRestGetDataPointsUsingURL(createParams(map[string]interface{}{
			"url":         client.GetGraphTokenSupply,
			"data-points": "1",
			"to":          to,
		}), http.StatusOK)

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return error when missing to param", func(t *testing.T) {
		t.Parallel()
		from := ""

		resp, httpResp, httpErr := apiClient.V1SCRestGetDataPointsUsingURL(createParams(map[string]interface{}{
			"url":         client.GetGraphTokenSupply,
			"data-points": "1",
			"from":        from,
		}), http.StatusOK)

		require.NotNil(t, httpErr)
		require.Equal(t, http.StatusBadRequest, httpResp.StatusCode())
		require.Nil(t, resp)
	})

	t.Run("test api should return successfully", func(t *testing.T) {
		t.Parallel()
		from := strconv.FormatInt(int64(math.Floor(float64(time.Now().Unix()-86400000/1000))), 10)
		to := strconv.FormatInt(int64(math.Floor(float64(time.Now().Unix()/1000))), 10)

		resp, httpResp, httpErr := apiClient.V1SCRestGetDataPointsUsingURL(createParams(map[string]interface{}{
			"url":         client.GetGraphTokenSupply,
			"data-points": "1",
			"from":        from,
			"to":          to,
		}), http.StatusOK)

		require.Nil(t, httpErr)
		require.Equal(t, http.StatusOK, httpResp.StatusCode())
		require.NotNil(t, resp)
		require.Equal(t, 1, len(resp))
	})
}
