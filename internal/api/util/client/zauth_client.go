package client

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/gosdk_common/core/transaction"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

type ZauthClient struct {
	BaseHttpClient
	zauthEntrypoint string
}

func NewZauthClient(zauthEntrypoint string) *ZauthClient {
	zauthClient := &ZauthClient{
		zauthEntrypoint: zauthEntrypoint,
	}
	zauthClient.HttpClient = resty.New()

	return zauthClient
}

func (c *ZauthClient) NewZauthHeaders(jwtToken, peerPublicKey string) map[string]string {
	zauthHeaders := map[string]string{
		"X-Jwt-Token":       jwtToken,
		"X-Peer-Public-Key": peerPublicKey,
	}

	return zauthHeaders
}

func (c *ZauthClient) Setup(t *test.SystemTest, setupWallet *model.SetupWallet, headers map[string]string) (*resty.Response, error) {
	t.Logf("performing setup of split wallet for jwt token [%v] using zauth...", headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/setup")

	var body []byte

	body, err = json.Marshal(setupWallet)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZauthClient) UpdateRestrictions(t *test.SystemTest, clientID string, restrictions []string, headers map[string]string) (*resty.Response, error) {
	t.Logf("update restrictions for client id [%v] for split key [%v] for peer public key [%v] and for jwt token [%v] using zvault...", clientID, restrictions, headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
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

func (c *ZauthClient) SignTransaction(t *test.SystemTest, signTransactionRequest *transaction.Transaction, headers map[string]string) (string, *resty.Response, error) {
	t.Logf("signing transaction for peer public key [%v] and for jwt token [%v] using zauth...", headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])
	var signTransactionResponse string

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/sign/txn")

	var body []byte

	body, err = json.Marshal(signTransactionRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &signTransactionResponse,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return signTransactionResponse, resp, err
}

func (c *ZauthClient) SignMessage(t *test.SystemTest, message *model.SignMessageRequest, headers map[string]string) (*model.SignMessageResponse, *resty.Response, error) {
	t.Logf("signing message for peer public key [%v] and for jwt token [%v] using zauth...", headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])
	var signMessageResponse *model.SignMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/sign/msg")

	var body []byte

	body, err = json.Marshal(message)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &signMessageResponse,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return signMessageResponse, resp, err
}

func (c *ZauthClient) Revoke(t *test.SystemTest, clientID, peerPublicKey string, headers map[string]string) (*resty.Response, error) {
	t.Logf("revoking split key including split wallet for client id [%v] and for peer public key [%v] and for jwt token [%v] using zauth...", clientID, peerPublicKey, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/revoke/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers: headers,
		FormData: map[string]string{
			"peer_public_key": peerPublicKey,
		},
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZauthClient) Delete(t *test.SystemTest, clientID string, headers map[string]string) (*resty.Response, error) {
	t.Logf("deleting split keys including split wallet for client id [%v] for jwt token [%v] using zauth...", clientID, headers["X-Jwt-Token"])

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/delete/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZauthClient) GetKeyDetails(t *test.SystemTest, clientID string, headers map[string]string) (*model.KeyDetailsResponse, *resty.Response, error) {
	t.Logf("get keys for client id [%v] peer public key [%v] and jwt token [%v] using zvault...", clientID, headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])
	var keyDetailsResponse *model.KeyDetailsResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath(fmt.Sprintf("/key/%s", clientID))

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &keyDetailsResponse,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return keyDetailsResponse, resp, err
}
