package client

import (
	"fmt"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
)

const (
	CreateBukectPath = "action=creatBucket&"
)

type ZS3Client struct {
	BaseHttpClient
	zssServerUrl string
}

func NewZS3Client(zssServerUrl string) *ZS3Client {
	zs3Client := &ZS3Client{}
	zs3Client.HttpClient = resty.New()
	zs3Client.zssServerUrl = zssServerUrl
	return zs3Client
}

func (c *ZS3Client) CreateBucket(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"action":          "createBucket",
		"accessKey":       "rootroot",
		"secretAccessKey": "rootroot",
		"bucketName":      "system-test1",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zssServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(resp.String())
	t.Logf("%s returned %s with status %s", c.zssServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) ListBucket(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"action":          "listBuckets",
		"accessKey":       "rootroot",
		"secretAccessKey": "rootroot",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zssServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(resp.String())
	t.Logf("%s returned %s with status %s", c.zssServerUrl, resp.String(), resp.Status())
	return resp, nil
}
