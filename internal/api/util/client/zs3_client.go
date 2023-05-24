package client

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"strings"

	"github.com/0chain/system_test/internal/api/util/test"
	resty "github.com/go-resty/resty/v2"
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

func (c *ZS3Client) BucketOperation(t *test.SystemTest, queryParams, formData map[string]string) (*resty.Response, error) {
	resp, err := c.BaseHttpClient.HttpClient.R().SetFiles(formData).SetQueryParams(queryParams).Post(c.zs3ServerUrl)
	if err != nil {
		t.Log(err)
		return nil, err
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}

func createForm(form map[string]string) (string, io.Reader, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()
	for key, val := range form {
		if strings.HasPrefix(val, "@") {
			val = val[1:]
			file, err := os.Open(val)
			if err != nil {
				return "", nil, err
			}
			defer file.Close() //nolint:gocritic
			part, err := mp.CreateFormFile(key, val)
			if err != nil {
				return "", nil, err
			}
			_, err = io.Copy(part, file)
			if err != nil {
				return "", nil, err
			}
		} else {
			err := mp.WriteField(key, val)
			if err != nil {
				return "", nil, err
			}
		}
	}
	return mp.FormDataContentType(), body, nil
}

func (c *ZS3Client) PutObject(t *test.SystemTest, queryParams, formData map[string]string) (*resty.Response, error) {
	ct, body, err := createForm(formData)
	if err != nil {
		t.Log(err)
		return nil, err
	}
	resp, err := c.BaseHttpClient.HttpClient.R().SetBody(body).SetHeaders(map[string]string{"Content-Type": ct}).SetQueryParams(queryParams).Get(c.zs3ServerUrl)
	if err != nil {
		t.Log(err)
		return nil, err
	}
	t.Logf("%s returned %s with status %s", c.zs3ServerUrl, resp.String(), resp.Status())
	return resp, nil
}
