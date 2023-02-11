package client

import (
	"fmt"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
)

const (
	AccessKey       = "rootroot"
	SecretAccessKey = "rootroot"
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

func (c *ZS3Client) CreateBucket(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "createBucket",
		"bucketName":      "system-test1",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(resp.String())
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) ListBucket(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "listBuckets",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) ListBucketsObjects(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "listBucketsObjects",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) ListObjects(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "listObjects",
		"bucketName":      "root",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) PutObject(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "putObject",
		"bucketName":      "root",
	}

	formData := map[string]string{
		"file": "https://github.com/0chain/zs3server/blob/task/logapi/assets/main-struture.png",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetFiles(formData).SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func (c *ZS3Client) RemoveObject(t *test.SystemTest) (*resty.Response, error) {
	queryParams := map[string]string{
		"accessKey":       AccessKey,
		"secretAccessKey": SecretAccessKey,
		"action":          "removeObject",
		"bucketName":      "root",
		"objectName":      "main-struture.png",
	}

	formData := map[string]string{
		"file": "https://github.com/0chain/zs3server/blob/task/logapi/assets/main-struture.png",
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetFiles(formData).SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}
