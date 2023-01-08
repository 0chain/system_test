package client

import (
	"fmt"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

type ZboxClient struct {
	BaseHttpClient
	zboxEntrypoint     string
	DefaultPhoneNumber string
}

func NewZboxClient(zboxEntrypoint string, defaultPhoneNumber string) *ZboxClient {
	zboxClient := &ZboxClient{
		zboxEntrypoint:     zboxEntrypoint,
		DefaultPhoneNumber: defaultPhoneNumber,
	}
	zboxClient.HttpClient = resty.New()

	return zboxClient
}

func (c *ZboxClient) FirebaseSendSms(t *test.SystemTest, firebaseKey, phoneNumber string) (*model.FirebaseSession, *resty.Response, error) {
	t.Logf("Sending firebase SMS...")
	var firebaseSession *model.FirebaseSession

	urlBuilder := NewURLBuilder().
		SetScheme("https").
		SetHost("identitytoolkit.googleapis.com").
		SetPath("/v1/accounts:sendVerificationCode").
		AddParams("key", firebaseKey)

	formData := map[string]string{
		"phoneNumber": phoneNumber,
		"appId":       "com.0Chain.0Box",
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &firebaseSession,
		FormData:           formData,
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return firebaseSession, resp, err
}

func (c *ZboxClient) FirebaseCreateToken(t *test.SystemTest, firebaseKey, sessionInfo string) (*model.FirebaseToken, *resty.Response, error) {
	t.Logf("Creating firebase token...")
	var firebaseToken *model.FirebaseToken

	urlBuilder := NewURLBuilder().
		SetScheme("https").
		SetHost("identitytoolkit.googleapis.com").
		SetPath("/v1/accounts:signInWithPhoneNumber").
		AddParams("key", firebaseKey)

	formData := map[string]string{
		"code":        "123456",
		"sessionInfo": sessionInfo,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &firebaseToken,
		FormData:           formData,
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return firebaseToken, resp, err
}

func (c *ZboxClient) CreateCSRFToken(t *test.SystemTest, phoneNumber string) (string, *resty.Response, error) {
	t.Logf("Creating CSRF Token using 0box...")
	var csrfToken *model.CSRFToken

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/csrftoken")
	parsedUrl := urlBuilder.String()

	resp, err := c.executeForServiceProvider(t, parsedUrl, model.ExecutionRequest{
		Dst:                &csrfToken,
		Headers:            map[string]string{"X-App-Phone-Number": phoneNumber},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return csrfToken.CSRFToken, resp, err
}

func (c *ZboxClient) ListWallets(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxWalletList, *resty.Response, error) {
	t.Logf("Listing all wallets for [%v] using 0box...", phoneNumber)
	var zboxWallets *model.ZboxWalletList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/list")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &zboxWallets,
		Headers: map[string]string{
			"X-App-ID-TOKEN":     idToken,
			"X-App-Phone-Number": phoneNumber,
			"X-CSRF-TOKEN":       csrfToken,
			"X-APP-TYPE":         "chimney",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxWallets, resp, err
}

func (c *ZboxClient) PostWallet(t *test.SystemTest, mnemonic, walletName, walletDescription, idToken, csrfToken, phoneNumber string) (*model.ZboxWalletAlt, *resty.Response, error) {
	t.Logf("Posting wallet using 0box...")
	var zboxWallet *model.ZboxWalletAlt

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	formData := map[string]string{
		"mnemonic":    mnemonic,
		"name":        walletName,
		"description": walletDescription,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxWallet,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "chimney",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return zboxWallet, resp, err
}

func (c *ZboxClient) DeleteWallet(t *test.SystemTest, walletId int, idToken, csrfToken, phoneNumber string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Deleting wallet id [%v] using 0box...", walletId)
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	formData := map[string]string{
		"wallet_id": fmt.Sprintf("%v", walletId),
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:  &message,
		Body: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "chimney",
		},
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)

	return message, resp, err
}

func (c *ZboxClient) ListWalletKeys(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (model.ZboxWalletKeys, *resty.Response, error) {
	t.Logf("Listing wallets keys for [%v] using 0box...", phoneNumber)
	var zboxWallets *model.ZboxWalletKeys

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/keys")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &zboxWallets,
		Headers: map[string]string{
			"X-App-Client-ID":    "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":   "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":    "1618213324",
			"X-App-ID-TOKEN":     idToken,
			"X-App-Phone-Number": phoneNumber,
			"X-CSRF-TOKEN":       csrfToken,
			"X-APP-TYPE":         "chimney",
		}, //FIXME: List endpoint does not require signature see: https://github.com/0chain/0box/issues/376
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return *zboxWallets, resp, err
}
