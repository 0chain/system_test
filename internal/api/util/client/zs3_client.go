package client

import (
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
)

type ZS3Client struct {
	BaseHttpClient
	zs3ServerUrl string
}

func NewZS3Client(zs3ServerUrl string) *ZS3Client {
	zs3Client := &ZS3Client{}
	zs3Client.HttpClient = resty.New()
	zs3Client.zs3ServerUrl = zs3ServerUrl
	return zs3Client
}

func (c *ZS3Client) BucketOperation(t *test.SystemTest, queryParams, formData map[string]string) (*resty.Response, error) {
	resp, err := c.BaseHttpClient.HttpClient.R().SetFiles(formData).SetQueryParams(queryParams).Post(c.zs3ServerUrl)
	if err != nil {
		t.Log(err)
		return nil, err
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}
