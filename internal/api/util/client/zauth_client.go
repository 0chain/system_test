package client

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/gosdk/core/transaction"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

const (
	X_PEER_PUBLIC_KEY = ""
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

func (c *ZauthClient) NewZauthHeaders() map[string]string {
	zauthHeaders := map[string]string{
		"X-User-ID":         X_USER_ID,
		"X-Jwt-Token":       X_JWT_TOKEN,
		"X-Peer-Public-Key": X_PEER_PUBLIC_KEY,
	}

	return zauthHeaders
}

func (c *ZauthClient) Setup(t *test.SystemTest, setupWallet model.SetupWallet, headers map[string]string) (*model.SetupResponse, *resty.Response, error) {
	t.Logf("performing setup of split wallet for jwt token [%v] using zauth...", headers["X-Jwt-Token"])
	var setupResponse *model.SetupResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/setup")

	var body []byte

	body, err = json.Marshal(setupWallet)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                setupResponse,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return setupResponse, resp, err
}

func (c *ZauthClient) SignTransaction(t *test.SystemTest, signTransactionRequest transaction.Transaction, headers map[string]string) (*transaction.Transaction, *resty.Response, error) {
	t.Logf("signing transaction for peer public key [%v] and for jwt token [%v] using zauth...", headers["X-Peer-Public-Key"], headers["X-Jwt-Token"])
	var signTransactionResponse *transaction.Transaction

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zauthEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/sign/txn")

	var body []byte

	body, err = json.Marshal(signTransactionRequest)
	require.NoError(t, err)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                signTransactionResponse,
		Headers:            headers,
		Body:               body,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return signTransactionResponse, resp, err
}

func (c *ZauthClient) SignMessage(t *test.SystemTest, message model.SignMessageRequest, headers map[string]string) (*model.SignMessageResponse, *resty.Response, error) {
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
		Dst:                signMessageResponse,
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
