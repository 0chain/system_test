package cliutils

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func addParms(url string, params map[string]string) string {
	first := true
	for key, value := range params {
		if first {
			url += "?"
			first = false
		} else {
			url += "&"
		}
		url += key + "=" + value
	}
	return url
}

func ApiGet[T any](t *testing.T, url string, params map[string]string) *T {
	url = addParms(url, params)

	res, err := http.Get(url)

	require.NoError(t, err, "with request", url)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request %s, status code: %d", url, res.StatusCode)
	require.NotNil(t, res.Body, "API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var result = new(T)
	err = json.Unmarshal(resBody, &result)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)
	return result
}

func ApiGetList[T any](t *testing.T, url string, params map[string]string, from, to int64) []T {
	var out []T
	var offset int64
	for {
		var temp []T
		raw := getNext(t, url, from, to, MaxQueryLimit, offset, params)

		err := json.Unmarshal(raw, &temp)
		require.NoError(t, err, "deserializing JSON string `%s`: %v", string(raw), err)

		offset += int64(len(temp))
		out = append(out, temp...)
		if len(temp) < MaxQueryLimit {
			return out
		}
	}
}

func getNext(t *testing.T, url string, from, to, limit, offset int64, params map[string]string) []byte {
	params["start"] = strconv.FormatInt(from, 10)
	params["end"] = strconv.FormatInt(to, 10)
	if limit > 0 {
		params["limit"] = strconv.FormatInt(limit, 10)
	}
	if offset > 0 {
		params["offset"] = strconv.FormatInt(offset, 10)
	}
	url = addParms(url, params)

	res, err := http.Get(url)

	require.NoError(t, err, "retrieving blocks %d to %d", from, to)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request to get blocks %d to %d, status code: %d", from, to, res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)
	return resBody
}
