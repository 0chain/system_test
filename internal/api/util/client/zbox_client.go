package client

import (
	"fmt"
	"strconv"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	resty "github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

type ZboxClient struct {
	BaseHttpClient
	zboxEntrypoint        string
	DefaultPhoneNumber    string
	DefaultMnemonic       string
	DefaultAllocationId   string
	DefaultAllocationName string
	DefaultAuthTicket     string
	DefaultRecieverId     string
}

func NewZboxClient(zboxEntrypoint, defaultPhoneNumber string) *ZboxClient {
	zboxClient := &ZboxClient{
		zboxEntrypoint:        zboxEntrypoint,
		DefaultPhoneNumber:    defaultPhoneNumber,
		DefaultAllocationName: "DefaultName",
		DefaultAllocationId:   "7df193bcbe12fc3ef9ff143b7825d9afadc3ce3d7214162f13ffad2510494d41",
		DefaultMnemonic:       "613ed9fb5b9311f6f22080eb1db69b2e786c990706c160faf1f9bdd324fd909bc640ad6a3a44cb4248ddcd92cc1fabf66a69ac4eb38a102b984b98becb0674db7d69c5727579d5f756bb8c333010866d4d871dae1b7032d6140db897e4349f60f94f1eb14a3b7a14a489226a1f35952472c9b2b13e3698523a8be2dcba91c344f55da17c21c403543d82fe5a32cb0c8133759ab67c31f1405163a2a255ec270b1cca40d9f236e007a3ba8f6be4eaeaad10376c5f224bad45c597d85a3b8b984f46c597f6cf561405bd0b0007ac6833cfff408aeb51c0d2fX",
		DefaultAuthTicket:     "eyJjbGllbnRfaWQiOiIiLCJvd25lcl9pZCI6ImEzMzQ1NGRhMTEwZGY0OTU2ZDc1YzgyMDA2N2M1ZThmZTJlZjIyZjZkNWQxODVhNWRjYTRmODYwMDczNTM1ZDEiLCJhbGxvY2F0aW9uX2lkIjoiZTBjMmNkMmQ1ZmFhYWQxM2ZjNTM3MzNkZDc1OTc0OWYyYjJmMDFhZjQ2MzMyMDA5YzY3ODIyMWEyYzQ4ODE1MyIsImZpbGVfcGF0aF9oYXNoIjoiZTcyNGEyMjAxZTIyNjUzZDMyMTY3ZmNhMWJmMTJiMmU0NGJhYzYzMzdkM2ViZGI3NDI3ZmJhNGVlY2FhNGM5ZCIsImFjdHVhbF9maWxlX2hhc2giOiIxZjExMjA4M2YyNDA1YzM5NWRlNTFiN2YxM2Y5Zjc5NWFhMTQxYzQwZjFkNDdkNzhjODNhNDk5MzBmMmI5YTM0IiwiZmlsZV9uYW1lIjoiSU1HXzQ4NzQuUE5HIiwicmVmZXJlbmNlX3R5cGUiOiJmIiwiZXhwaXJhdGlvbiI6MCwidGltZXN0YW1wIjoxNjY3MjE4MjcwLCJlbmNyeXB0ZWQiOmZhbHNlLCJzaWduYXR1cmUiOiIzMzllNTUyOTliNDhlMjI5ZGRlOTAyZjhjOTY1ZDE1YTk0MGIyNzc3YzVkOTMyN2E0Yzc5MTMxYjhhNzcxZTA3In0=",
		DefaultRecieverId:     "a33454da110df4956d75c820067c5e8fe2ef22f6d5d185a5dca4f860073535d1",
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

func (c *ZboxClient) CreateCSRFToken(t *test.SystemTest, phoneNumber string) (*model.CSRFToken, *resty.Response, error) {
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

	return csrfToken, resp, err
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
			"X-APP-TYPE":         "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxWallets, resp, err
}

func (c *ZboxClient) GetAllocation(t *test.SystemTest, idToken, csrfToken, phoneNumber, allocationId, allocationName string) (model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocationId)
	var allocation model.ZboxAllocation

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	formData := map[string]string{
		"id":   allocationId,
		"name": allocationName,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &allocation,
		QueryParams: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return allocation, resp, err
}

func (c *ZboxClient) ListAllocation(t *test.SystemTest, idToken, csrfToken, phoneNumber string) ([]model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Listing all allocations for [%v] using 0box...", phoneNumber)
	var allocWalletList []model.ZboxAllocation

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
			"X-APP-TYPE":             "blimp",
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
			"X-App-Type":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return zboxWallet, resp, err
}

func (c *ZboxClient) PostAllocation(t *test.SystemTest, allocationId, allocationName, allocationDescription, allocationType, idToken, csrfToken, phoneNumber string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Posting Allocation using 0box...")
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	formData := map[string]string{
		"name":            allocationName,
		"id":              allocationId,
		"description":     allocationDescription,
		"allocation_type": allocationType,
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
			"X-APP-TYPE":         "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)
	return message, resp, err
}

func (c *ZboxClient) UpdateAllocation(t *test.SystemTest, allocationId, allocationName, allocationDescription, allocationType, idToken, csrfToken, phoneNumber string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Posting Allocation using 0box...")
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	formData := map[string]string{
		"name":            allocationName,
		"id":              allocationId,
		"description":     allocationDescription,
		"allocation_type": allocationType,
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
			"X-APP-TYPE":         "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPUTMethod)
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
			"X-APP-TYPE":             "blimp",
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
			"X-APP-TYPE":         "blimp",
		}, //FIXME: List endpoint does not require signature see: https://github.com/0chain/0box/issues/376
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return *zboxWallets, resp, err
}

func (c *ZboxClient) PostUserInfoBiography(t *test.SystemTest, bio, idToken, csrfToken, phoneNumber string) (*model.ZboxSuccess, *resty.Response, error) {
	t.Logf("Posting user info biography using 0box...")
	var zboxSuccess *model.ZboxSuccess

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/bio")

	formData := map[string]string{
		"biography": bio,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxSuccess,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return zboxSuccess, resp, err
}

func (c *ZboxClient) PostUserInfoAvatar(t *test.SystemTest, filePath, idToken, csrfToken, phoneNumber string) (*model.ZboxSuccess, *resty.Response, error) {
	t.Logf("Posting user info avatar using 0box...")
	var zboxSuccess *model.ZboxSuccess

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/avatar")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxSuccess,
		FileName: "avatar",
		FilePath: filePath,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpFileUploadMethod)

	return zboxSuccess, resp, err
}

func (c *ZboxClient) PostUserInfoBackgroundImage(t *test.SystemTest, filePath, idToken, csrfToken, phoneNumber string) (*model.ZboxSuccess, *resty.Response, error) {
	t.Logf("Posting user info background using 0box...")
	var zboxSuccess *model.ZboxSuccess

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/bgimg")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxSuccess,
		FileName: "background",
		FilePath: filePath,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpFileUploadMethod)

	return zboxSuccess, resp, err
}

func (c *ZboxClient) GetUserInfo(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxUserInfo, *resty.Response, error) {
	t.Logf("Getting user info using 0box...")
	var userInfo *model.ZboxUserInfo

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo")

	formData := map[string]string{
		"phone_number": phoneNumber,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &userInfo,
		FormData: formData,
		Headers: map[string]string{
			"X-CSRF-TOKEN": csrfToken,
			"X-APP-TYPE":   "blimp",
		}, // TODO: this endpoint doesnt check signature!
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return userInfo, resp, err
}

func (c *ZboxClient) GetUserInfoFromUserName(t *test.SystemTest, idToken, csrfToken, userName string) (*model.ZboxUserInfo, *resty.Response, error) {
	t.Logf("Getting user info using 0box...")
	var userInfo *model.ZboxUserInfo

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo")

	formData := map[string]string{
		"username": userName,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &userInfo,
		FormData: formData,
		Headers: map[string]string{
			"X-CSRF-TOKEN": csrfToken,
			"X-APP-TYPE":   "blimp",
		}, // TODO: this endpoint doesnt check signature!
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return userInfo, resp, err
}

func (c *ZboxClient) PutUsername(t *test.SystemTest, username, idToken, csrfToken, phoneNumber string) (*model.ZboxUsername, *resty.Response, error) {
	t.Logf("Putting username using 0box...")
	var zboxUsername *model.ZboxUsername

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/username")

	formData := map[string]string{
		"username": username,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxUsername,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return zboxUsername, resp, err
}

func (c *ZboxClient) GetShareInfo(t *test.SystemTest, idToken, csrfToken, phoneNumber, shareMessage, fromInfo, authTickets, recieverClientId string) (model.ZboxShareInfoList, *resty.Response, error) {
	t.Logf("Getting share Info for  authentication ticket [%v] using 0box...", authTickets[0])
	var ZboxShareInfoList model.ZboxShareInfoList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/share/shareinfo")

	formData := map[string]string{
		"auth_tickets":       authTickets,
		"message":            shareMessage,
		"from_info":          fromInfo,
		"receiver_client_id": recieverClientId,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxShareInfoList,
		QueryParams: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxShareInfoList, resp, err
}

func (c *ZboxClient) PostShareInfo(t *test.SystemTest, authTicket, shareMessage, fromInfo, recieverClientId, idToken, csrfToken, phoneNumber string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Posting ShareInfo using 0box...")
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/share/shareinfo")

	formData := map[string]string{
		"auth_ticket":        authTicket,
		"message":            shareMessage,
		"from_info":          fromInfo,
		"receiver_client_id": recieverClientId,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &message,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)
	return message, resp, err
}

func (c *ZboxClient) DeleteShareInfo(t *test.SystemTest, idToken, csrfToken, phoneNumber, authTicket string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("Deleting shareInfo for auth_ticket [%v] using 0box...", authTicket)
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/share/shareinfo")

	formData := map[string]string{
		"auth_ticket": authTicket,
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
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)

	return message, resp, err
}
func (c *ZboxClient) GetWalletKeys(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxWalletKeys, *resty.Response, error) {
	t.Logf("Getting wallet keys for [%v] using 0box...", phoneNumber)
	var zboxWalletKeys *model.ZboxWalletKeys

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/keys")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &zboxWalletKeys,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)
	return zboxWalletKeys, resp, err
}

func (c *ZboxClient) UpdateWallet(t *test.SystemTest, mnemonic, walletName, walletDescription, idToken, csrfToken, phoneNumber string) (*model.ZboxWalletAlt, *resty.Response, error) {
	t.Logf("Updating wallet using 0box...")
	var zboxWallet *model.ZboxWalletAlt

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	formData := map[string]string{
		"mnemonic":    mnemonic,
		"allocation":  "",
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
			"X-APP-TYPE":             "blimp",
		},
	}, HttpPUTMethod)

	return zboxWallet, resp, err
}

func (c *ZboxClient) ContactWallet(t *test.SystemTest, reqBody, idToken, csrfToken, phoneNumber string) (*resty.Response, error) {
	t.Logf("Contacting wallets for [%v] using 0box...", phoneNumber)

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/contact/wallets")

	formData := map[string]string{
		"contacts": reqBody,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &model.MessageContainer{},
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":    "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":   "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Timestamp":    "1618213324",
			"X-App-ID-TOKEN":     idToken,
			"X-App-Phone-Number": phoneNumber,
			"X-CSRF-TOKEN":       csrfToken,
			"X-APP-TYPE":         "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZboxClient) CreateNftCollectionId(t *test.SystemTest, idToken, csrfToken, phoneNumber, createdBy, collectionName, collectionId, totalNfts, collectionType, allocationId, baseUrl, symbol string, pricePerPack, maxMints, CurrMints, batchSize int) (*model.ZboxNftCollection, *resty.Response, error) {
	t.Logf("Creating nft collection using 0box...")
	var ZboxNftCollection *model.ZboxNftCollection

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	formData := map[string]string{
		"createdBy":       createdBy,
		"collection_name": collectionName,
		"collection_id":   collectionId,
		"total_nfts":      totalNfts,
		"collection_type": collectionType,
		"allocation_id":   allocationId,
		"base_url":        baseUrl,
		"symbol":          symbol,
		"price_per_pack":  strconv.Itoa(pricePerPack),
		"max_mints":       strconv.Itoa(maxMints),
		"curr_mints":      strconv.Itoa(CurrMints),
		"batch_size":      strconv.Itoa(batchSize),
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxNftCollection,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31ftring(pricePerPack)740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return ZboxNftCollection, resp, err
}

func (c *ZboxClient) PostNftCollection(t *test.SystemTest, idToken, csrfToken, phoneNumber, stage_nft_upload, nft_reference, collectionId, owned_by, nft_activity, meta_data, allocationId, created_by, contract_address, token_id, token_standard string) (*model.ZboxNft, *resty.Response, error) {
	t.Logf("Posting nft using 0box...")
	var ZboxNft *model.ZboxNft

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft")

	formData := map[string]string{
		"stage":            stage_nft_upload,
		"reference":        nft_reference,
		"collection_id":    collectionId,
		"owned_by":         owned_by,
		"nft_activity":     nft_activity,
		"meta_data":        meta_data,
		"allocation_id":    allocationId,
		"created_by":       created_by,
		"contract_address": contract_address,
		"token_id":         token_id,
		"token_standard":   token_standard,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxNft,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31ftring(pricePerPack)740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return ZboxNft, resp, err
}

func (c *ZboxClient) GetAllNftCollection(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxNftList, *resty.Response, error) {
	t.Logf("Getting All nft using 0box...")
	var ZboxNftList *model.ZboxNftList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/all")

	queryParams := map[string]string{}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNftList,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        "31ftring(pricePerPack)740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96",
			"X-App-Client-Signature": "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a",
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-Phone-Number":     phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftList, resp, err
}
