package client

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

const (
	X_USER_ID   = "lWVZRhERosYtXR9MBJh5yJUtweI3"
	X_JWT_TOKEN = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoibFdWWlJoRVJvc1l0WFI5TUJKaDV5SlV0d2VJMyIsImV4cCI6MTcyMjc2OTc5M30.QW1GUdGVi2NERMyrkdIwFNXmZ7ZInaB2er5gY6zOv_xEe8NmmZvn5tGFk2Agc8A1TraPfYWvvnMPvtNu8U6tiA"
)

type ZvaultClient struct {
	BaseHttpClient
	zvaultEntrypoint string
}

func NewZvaultClient(zvaultEntrypoint string) *ZvaultClient {
	zvaultClient := &ZvaultClient{
		zvaultEntrypoint: zvaultEntrypoint,
	}
	zvaultClient.HttpClient = resty.New()

	return zvaultClient
}

func (c *ZvaultClient) NewZvaultHeaders() map[string]string {
	zvaultHeaders := map[string]string{
		"X-User-ID":   X_USER_ID,
		"X-Jwt-Token": X_JWT_TOKEN,
	}

	return zvaultHeaders
}

func (c *ZvaultClient) GenerateSplitWallet(t *test.SystemTest, headers map[string]string) (*model.SplitWallet, *resty.Response, error) {
	t.Logf("generating new split wallet for jwt token [%v] using zvault...", headers["X-Jwt-Token"])
	var splitWallet *model.SplitWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/generate")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                splitWallet,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return splitWallet, resp, err
}

func (c *ZvaultClient) GenerateSplitKey(t *test.SystemTest, clientID string, headers map[string]string) (*model.SplitWallet, *resty.Response, error) {
	t.Logf("generating new split key for client id [%v] and for jwt token [%v] using zvault...", clientID, headers["X-Jwt-Token"])
	var splitKey *model.SplitWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/generate/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                splitKey,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return splitKey, resp, err
}

func (c *ZvaultClient) Store(t *test.SystemTest, privateKey string, headers map[string]string) (*model.SplitWallet, *resty.Response, error) {
	t.Logf("storing private key [%v] and for jwt token [%v] using zvault...", privateKey, headers["X-Jwt-Token"])
	var splitKey *model.SplitWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/store")

	storeRequest := &model.StoreRequest{
		PrivateKey: privateKey,
	}

	var body []byte

	body, err = json.Marshal(storeRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                splitKey,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return splitKey, resp, err
}

func (c *ZvaultClient) GetKeys(t *test.SystemTest, clientID string, headers map[string]string) ([]*model.SplitKey, *resty.Response, error) {
	t.Logf("get keys for client id [%v] and for jwt token [%v] using zvault...", clientID, headers["X-Jwt-Token"])
	var splitKey []*model.SplitKey

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/keys/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &splitKey,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return splitKey, resp, err
}

func (c *ZvaultClient) Revoke(t *test.SystemTest, clientID, publicKey string, headers map[string]string) (*resty.Response, error) {
	t.Logf("revoking split key including split wallet for client id [%v] and for public key [%v] and for jwt token [%v] using zvault...", clientID, publicKey, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/revoke/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers: headers,
		FormData: map[string]string{
			"public_key": publicKey,
		},
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZvaultClient) Delete(t *test.SystemTest, clientID string, headers map[string]string) (*resty.Response, error) {
	t.Logf("deleting split keys including split wallet for client id [%v] for jwt token [%v] using zvault...", clientID, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/delete/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZvaultClient) GetWallets(t *test.SystemTest, headers map[string]string) ([]*model.SplitKey, *resty.Response, error) {
	t.Logf("get wallets for jwt token [%v] using zvault...", headers["X-Jwt-Token"])
	var splitKey []*model.SplitKey

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/wallets")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &splitKey,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return splitKey, resp, err
}

func (c *ZvaultClient) ShareWallet(t *test.SystemTest, clientID, publicKey string, headers map[string]string) (*model.SplitWallet, *resty.Response, error) {
	t.Logf("sharing wallet with public key [%v] for client id [%v] and for jwt token [%v] using zvault...", publicKey, clientID, headers["X-Jwt-Token"])
	var splitKey *model.SplitWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/share")

	storeRequest := &model.ShareWalletRequest{
		PublicKey:    publicKey,
		TargetUserID: clientID,
	}

	var body []byte

	body, err = json.Marshal(storeRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                splitKey,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return splitKey, resp, err
}

func (c *ZvaultClient) GetSharedWallets(t *test.SystemTest, headers map[string]string) (*model.SplitWallet, *resty.Response, error) {
	t.Logf("retrieving shared wallets for jwt token [%v] using zvault...", headers["X-Jwt-Token"])
	var splitKey *model.SplitWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/wallets/shared")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                splitKey,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return splitKey, resp, err
}
