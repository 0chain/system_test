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
	zboxEntrypoint      string
	DefaultPhoneNumber  string
	DefaultMnemonic     string
	DefaultAllocationId string
}

func NewZboxClient(zboxEntrypoint, defaultPhoneNumber string) *ZboxClient {
	zboxClient := &ZboxClient{
		zboxEntrypoint:      zboxEntrypoint,
		DefaultPhoneNumber:  defaultPhoneNumber,
		DefaultAllocationId: "7df193bcbe12fc3ef9ff143b7825d9afadc3ce3d7214162f13ffad2510494d41",
		DefaultMnemonic:     "613ed9fb5b9311f6f22080eb1db69b2e786c990706c160faf1f9bdd324fd909bc640ad6a3a44cb4248ddcd92cc1fabf66a69ac4eb38a102b984b98becb0674db7d69c5727579d5f756bb8c333010866d4d871dae1b7032d6140db897e4349f60f94f1eb14a3b7a14a489226a1f35952472c9b2b13e3698523a8be2dcba91c344f55da17c21c403543d82fe5a32cb0c8133759ab67c31f1405163a2a255ec270b1cca40d9f236e007a3ba8f6be4eaeaad10376c5f224bad45c597d85a3b8b984f46c597f6cf561405bd0b0007ac6833cfff408aeb51c0d2fX",
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

func (c *ZboxClient) ListAllocation(t *test.SystemTest, idToken, csrfToken, phoneNumber string) ([]model.Allocationobj, *resty.Response, error) {
	t.Logf("Posting wallet using 0box...")
	var allocWalletList []model.Allocationobj

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation/list")

	formData := map[string]string{}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &allocWalletList,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "chimney",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return allocWalletList, resp, err
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

func (c *ZboxClient) PostAllocation(t *test.SystemTest, allocationId, allocationName, idToken, csrfToken, phoneNumber string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Posting Allocation using 0box...")
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	formData := map[string]string{
		"name": allocationName,
		"id":   allocationId,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &message,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":    "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":   "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":    "1618213324",
			"X-App-ID-TOKEN":     idToken,
			"X-App-Phone-Number": phoneNumber,
			"X-CSRF-TOKEN":       csrfToken,
			"X-APP-TYPE":         "chimney",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)
	return message, resp, err
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
