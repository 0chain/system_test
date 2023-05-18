package client

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/api/model"
	resty "github.com/go-resty/resty/v2"
)

// Statuses of http based responses
const (
	HttpOkStatus          = 200
	HttpBadRequestStatus  = 400
	HttpNotFoundStatus    = 404
	HttpNotModifiedStatus = 304
)

// Contains all methods used for http based requests
const (
	HttpPOSTMethod = iota + 1
	HttpGETMethod
	HttpPUTMethod
	HttpDELETEMethod
	HttpFileUploadMethod
)

type BaseHttpClient struct {
	HttpClient *resty.Client //nolint
}

func (c *BaseHttpClient) executeForServiceProvider(t *test.SystemTest, url string, executionRequest model.ExecutionRequest, method int) (*resty.Response, error) { //nolint
	var (
		resp *resty.Response
		err  error
	)

	switch method {
	case HttpPUTMethod:
		resp, err = c.HttpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetQueryParams(executionRequest.QueryParams).SetBody(executionRequest.Body).Put(url)
	case HttpPOSTMethod:
		resp, err = c.HttpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetBody(executionRequest.Body).Post(url)
	case HttpFileUploadMethod:
		resp, err = c.HttpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetFile(executionRequest.FileName, executionRequest.FilePath).Post(url)
	case HttpGETMethod:
		resp, err = c.HttpClient.R().SetHeaders(executionRequest.Headers).SetQueryParams(executionRequest.QueryParams).Get(url)
	case HttpDELETEMethod:
		resp, err = c.HttpClient.R().SetHeaders(executionRequest.Headers).SetFormData(executionRequest.FormData).SetBody(executionRequest.Body).Delete(url)
	}

	if err != nil {
		t.Errorf("%s error : %v", url, err)
		return nil, fmt.Errorf("%s: %w", url, ErrGetFromResource)
	}

	body := resp.Body()
	if executionRequest.Dst != nil {
		err = json.Unmarshal(body, executionRequest.Dst)
		if err != nil {
			t.Logf("%s returned %s with status %s", url, string(body), resp.Status())
			return resp, err
		}
	}

	return resp, nil
}
