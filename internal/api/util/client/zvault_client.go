package client

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
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

func (c *ZvaultClient) NewZvaultHeaders(jwtToken string) map[string]string {
	zvaultHeaders := map[string]string{
		"X-Jwt-Token": jwtToken,
	}

	return zvaultHeaders
}

func (c *ZvaultClient) GenerateSplitWallet(t *test.SystemTest, headers map[string]string) (*model.GenerateWalletResponse, *resty.Response, error) {
	t.Logf("generating new split wallet for jwt token [%v] using zvault...", headers["X-Jwt-Token"])

	var generateWalletResponse *model.GenerateWalletResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/wallet")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &generateWalletResponse,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return generateWalletResponse, resp, err
}

func (c *ZvaultClient) GenerateSplitKey(t *test.SystemTest, clientID string, headers map[string]string) (*resty.Response, error) {
	t.Logf("generating new split key for client id [%v] and for jwt token [%v] using zvault...", clientID, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/key/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZvaultClient) Store(t *test.SystemTest, privateKey, mnemonic string, headers map[string]string) (*resty.Response, error) {
	t.Logf("storing private key [%v], mnemonic [%v] and for jwt token [%v] using zvault...", privateKey, mnemonic, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/store")

	storeRequest := &model.StoreRequest{
		Mnemonic:   mnemonic,
		PrivateKey: privateKey,
	}

	var body []byte

	body, err = json.Marshal(storeRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZvaultClient) UpdateRestrictions(t *test.SystemTest, clientID string, restrictions []string, headers map[string]string) (*resty.Response, error) {
	t.Logf("update restrictions for split key [%v] for client id [%v] for peer public key [%v] and for jwt token [%v] using zvault...", restrictions, clientID, headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/restrictions/%s", clientID))

	updateRestrictionRequest := &model.UpdateRestrictionsRequest{
		Restrictions: restrictions,
	}

	var body []byte

	body, err = json.Marshal(updateRestrictionRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPUTMethod)

	return resp, err
}

func (c *ZvaultClient) ShareWallet(t *test.SystemTest, userID, publicKey string, headers map[string]string) (*resty.Response, error) {
	t.Logf("sharing wallet with public key [%v] for user id [%v] and for jwt token [%v] using zvault...", publicKey, userID, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/share")

	storeRequest := &model.ShareWalletRequest{
		PublicKey:    publicKey,
		TargetUserID: userID,
	}

	var body []byte

	body, err = json.Marshal(storeRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
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

func (c *ZvaultClient) GetRestrictions(t *test.SystemTest, headers map[string]string) ([]string, *resty.Response, error) {
	t.Logf("get keys for peer public key [%v] and jwt token [%v] using zvault...", headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])
	var restrictions []string

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/restrictions")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &restrictions,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return restrictions, resp, err
}

func (c *ZvaultClient) GetKeys(t *test.SystemTest, clientID string, headers map[string]string) (*model.GetKeyResponse, *resty.Response, error) {
	t.Logf("get keys for client id [%v] and for jwt token [%v] using zvault...", clientID, headers["X-Jwt-Token"])
	var keys *model.GetKeyResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/keys/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &keys,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return keys, resp, err
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

func (c *ZvaultClient) GetSharedWallets(t *test.SystemTest, headers map[string]string) ([]*model.SplitKey, *resty.Response, error) {
	t.Logf("retrieving shared wallets for jwt token [%v] using zvault...", headers["X-Jwt-Token"])
	var splitKeys []*model.SplitKey

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zvaultEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/wallets/shared")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &splitKeys,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return splitKeys, resp, err
}
