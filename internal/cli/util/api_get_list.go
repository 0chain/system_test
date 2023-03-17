package cliutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ApiGetRetries[T any](t *test.SystemTest, url string, params map[string]string, retries int) *T {
	var err error
	var res *T
	for try := 1; try <= retries; try++ {
		res, err = ApiGetError[T](url, params)
		if err != nil {
			t.Logf("retry %d, %v", try, err)
		} else {
			break
		}
	}
	assert.NoError(t, err, "%s failed after %d retries", url, retries)

	return res
}

func ApiGetError[T any](url string, params map[string]string) (*T, error) {
	url = addParms(url, params)

	res, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("with request %s, %v", url, err)
	}

	defer res.Body.Close()
	if res.StatusCode < 200 && res.StatusCode >= 300 {
		return nil, fmt.Errorf("failed API request %s, status code: %d: %v", url, res.StatusCode, err)
	}
	if res.Body == nil {
		return nil, fmt.Errorf("request %s, API response must not be nil", url)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("response %s, reading response body: %v", url, err)
	}

	var result = new(T)
	err = json.Unmarshal(resBody, &result)
	if err != nil {
		return nil, fmt.Errorf("deserializing JSON string `%s`: %v", string(resBody), err)
	}
	return result, nil
}

func ApiGet[T any](t *test.SystemTest, url string, params map[string]string) *T {
	url = addParms(url, params)
	t.Logf("api query %s ...", url)
	res, err := http.Get(url) //nolint:gosec

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

func ApiGetSlice[T any](t *test.SystemTest, url string, params map[string]string) []T {
	url = addParms(url, params)
	t.Logf("api query %s ...", url)
	res, err := http.Get(url) //nolint:gosec

	require.NoError(t, err, "with request", url)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request %s, status code: %d", url, res.StatusCode)
	require.NotNil(t, res.Body, "API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)

	var result []T
	err = json.Unmarshal(resBody, &result)
	require.NoError(t, err, "deserializing JSON string `%s`: %v", string(resBody), err)
	return result
}

func ApiGetList[T any](t *test.SystemTest, url string, params map[string]string, from, to int64) []T {
	var out []T
	var offset int64
	for {
		var temp []T
		raw := getNext(t, url, from, to, MaxQueryLimit, offset, params)

		err := json.Unmarshal(raw, &temp)
		assert.NoError(t, err, "deserializing JSON string `%s`: %v", string(raw), err)
		out = append(out, temp...)
		if len(temp) < MaxQueryLimit {
			return out
		}

		offset += int64(len(temp))
	}
}

func getNext(t *test.SystemTest, url string, from, to, limit, offset int64, params map[string]string) []byte {
	params["start"] = strconv.FormatInt(from, 10)
	params["end"] = strconv.FormatInt(to, 10)
	if limit > 0 {
		params["limit"] = strconv.FormatInt(limit, 10)
	}
	if offset > 0 {
		params["offset"] = strconv.FormatInt(offset, 10)
	}
	url = addParms(url, params)
	t.Logf("api list query %s ...", url)
	res, err := http.Get(url) //nolint:gosec

	require.NoError(t, err, "retrieving blocks %d to %d", from, to)
	defer res.Body.Close()
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300,
		"failed API request to get blocks %d to %d, status code: %d", from, to, res.StatusCode)
	require.NotNil(t, res.Body, "balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err, "reading response body: %v", err)
	return resBody
}

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
