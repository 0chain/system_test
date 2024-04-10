package client

import (
	"fmt"
	"strconv"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	resty "github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

const (
	X_APP_USER_ID            = "test_user_id"
	X_APP_ID_TOKEN           = "test_firebase_token"
	X_APP_CLIENT_ID          = "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9"
	X_APP_CLIENT_KEY         = "b6d86a895b9ab247b9d19280d142ffb68c3d89833db368d9a2ee9346fa378a05441635a5951d2f6a209c9ca63dc903353739bfa8ba79bad17690fe8e38622e96"
	X_APP_CLIENT_SIGNATURE   = "d903d0f57c96b052d907afddb62777a1f77a147aee5ed2b5d8bab60a9319b09a"
	X_APP_CLIENT_ID_R        = "3fb9694ebf47b5a51c050025d9c807c3319a05499b1eb980bbb9f1e27e119c9f"
	X_APP_CLIENT_KEY_R       = "9a8a960db2dd93eb35f26e8f7e84976349064cae3246da23abd575f05e7ed31bd90726cfcc960e017a9246d080f5419ada219d03758c370208c5b688e5ec7a9c"
	X_APP_CLIENT_SIGNATURE_R = "6b710d015b9e5e4734c08ac2de79ffeeeb49e53571cce8f71f21e375e5eca916"
	X_APP_TIMESTAMP          = "123456789"
	X_APP_CSRF               = "test_csrf_token"
	X_APP_BLIMP              = "blimp"
	X_APP_CHIMNEY            = "chimney"
	X_APP_VULT               = "vult"
	X_APP_BOLT               = "bolt"
	X_APP_CHALK              = "chalk"
)

type ZboxClient struct {
	BaseHttpClient
	zboxEntrypoint string
}

func NewZboxClient(zboxEntrypoint string) *ZboxClient {
	zboxClient := &ZboxClient{
		zboxEntrypoint: zboxEntrypoint,
	}
	zboxClient.HttpClient = resty.New()

	return zboxClient
}

func (c *ZboxClient) NewZboxHeaders(appType string) map[string]string {
	zboxHeaders := map[string]string{
		"X-App-Client-ID":        X_APP_CLIENT_ID,
		"X-App-Client-Key":       X_APP_CLIENT_KEY,
		"X-App-Timestamp":        X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         X_APP_ID_TOKEN,
		"X-App-User-ID":          X_APP_USER_ID,
		"X-CSRF-TOKEN":           X_APP_CSRF,
		"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
		"X-APP-TYPE":             appType,
	}
	return zboxHeaders
}

func (c *ZboxClient) NewZboxHeaders_R(appType string) map[string]string {
	zboxHeaders := map[string]string{
		"X-App-Client-ID":        X_APP_CLIENT_ID_R,
		"X-App-Client-Key":       X_APP_CLIENT_KEY_R,
		"X-App-Timestamp":        X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         X_APP_ID_TOKEN,
		"X-App-User-ID":          X_APP_USER_ID,
		"X-CSRF-TOKEN":           X_APP_CSRF,
		"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE_R,
		"X-APP-TYPE":             appType,
	}
	return zboxHeaders
}

func (c *ZboxClient) CreateOwner(t *test.SystemTest, headers, owner map[string]string) (*model.ZboxOwner, *resty.Response, error) {
	t.Logf("creating owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxOwner *model.ZboxOwner

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/owner")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxOwner,
		FormData:           owner,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)
	return zboxOwner, resp, err
}

func (c *ZboxClient) UpdateOwner(t *test.SystemTest, headers, owner map[string]string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("updating owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/owner")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &message,
		FormData:           owner,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)
	return message, resp, err
}

func (c *ZboxClient) GetOwner(t *test.SystemTest, headers map[string]string) (*model.ZboxOwner, *resty.Response, error) {
	t.Logf("getting owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxOwner *model.ZboxOwner

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/owner")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxOwner,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxOwner, resp, err
}

func (c *ZboxClient) DeleteOwner(t *test.SystemTest, headers map[string]string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("deleting owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/owner")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &message,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)

	return message, resp, err
}

func (c *ZboxClient) CreateWallet(t *test.SystemTest, headers, wallet map[string]string) (*model.ZboxWallet, *resty.Response, error) {
	t.Logf("creating wallet for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxWallet *model.ZboxWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxWallet,
		FormData:           wallet,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return zboxWallet, resp, err
}

func (c *ZboxClient) UpdateWallet(t *test.SystemTest, headers, wallet map[string]string) (*model.MessageContainer, *resty.Response, error) {
	t.Logf("updating wallet for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.MessageContainer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &message,
		FormData:           wallet,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return message, resp, err
}

func (c *ZboxClient) GetWalletKeys(t *test.SystemTest, headers map[string]string) (*model.ZboxWallet, *resty.Response, error) {
	t.Logf("getting wallet keys for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxWallet *model.ZboxWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/keys")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxWallet,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxWallet, resp, err
}

func (c *ZboxClient) GetWalletList(t *test.SystemTest, headers map[string]string) (*model.ZboxWalletList, *resty.Response, error) {
	t.Logf("getting wallet list for userID [%v] using 0box...", headers["X-App-User-ID"])
	var res *model.ZboxWalletList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/list")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &res,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return res, resp, err
}

func (c *ZboxClient) CreateAllocation(t *test.SystemTest, headers, allocation map[string]string) (model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocation["id"])
	var zboxAllocation model.ZboxAllocation

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxAllocation,
		FormData:           allocation,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return zboxAllocation, resp, err
}

func (c *ZboxClient) UpdateAllocation(t *test.SystemTest, headers, allocation map[string]string) (model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocation["id"])
	var zboxAllocation model.ZboxAllocation

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxAllocation,
		FormData:           allocation,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return zboxAllocation, resp, err
}

func (c *ZboxClient) GetAllocation(t *test.SystemTest, headers map[string]string, allocationID string) (model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocationID)
	var zboxAllocation model.ZboxAllocation

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &zboxAllocation,
		QueryParams: map[string]string{
			"id": allocationID,
		},
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxAllocation, resp, err
}

func (c *ZboxClient) ListAllocation(t *test.SystemTest, headers map[string]string) ([]model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocations for  userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxAllocation []model.ZboxAllocation

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation/list")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxAllocation,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxAllocation, resp, err
}

func (c *ZboxClient) CreateFreeStorage(t *test.SystemTest, headers map[string]string) (*model.ZboxFreeStorage, *resty.Response, error) {
	t.Logf("Creating FreeStorage using 0box...")
	var ZboxFreeStorage *model.ZboxFreeStorage

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/freestorage")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxFreeStorage,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxFreeStorage, resp, err
}

func (c *ZboxClient) GetDexState(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst: &dexState,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return dexState, resp, err
}

func (c *ZboxClient) PostDexState(t *test.SystemTest, data map[string]string, idToken, csrfToken, phoneNumber string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	formData := data

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst:      &dexState,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return dexState, resp, err
}

func (c *ZboxClient) PutDexState(t *test.SystemTest, data map[string]string, idToken, csrfToken, phoneNumber string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	formData := data

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst:      &dexState,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return dexState, resp, err
}

func (c *ZboxClient) CheckFundingStatus(t *test.SystemTest, fundingId, idToken, csrfToken, phoneNumber, appType string) (*model.ZboxFundingResponse, *resty.Response, error) {
	t.Logf("Checking status of funding using funding id")
	var zboxFundingResponse *model.ZboxFundingResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/zbox/fund")

	url := fmt.Sprintf("%s/%s", urlBuilder.String(), fundingId)
	resp, err := c.executeForServiceProvider(t, url, model.ExecutionRequest{
		Dst: &zboxFundingResponse,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Type":             appType,
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxFundingResponse, resp, err
}

func (c *ZboxClient) PostAllocation(t *test.SystemTest, allocationId, allocationName, allocationDescription, allocationType, idToken, csrfToken, phoneNumber, appType string) (*model.MessageContainer, *resty.Response, error) {
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
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-Timestamp":  "1618213324",
			"X-App-ID-TOKEN":   idToken,
			"X-App-User-ID":    phoneNumber,
			"X-CSRF-TOKEN":     csrfToken,
			"X-APP-TYPE":       appType,
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)
	return message, resp, err
}

func (c *ZboxClient) PostUserInfoBiography(t *test.SystemTest, bio, idToken, csrfToken, phoneNumber string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting user info biography using 0box...")
	var ZboxMessageResponse *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/bio")

	formData := map[string]string{
		"biography": bio,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxMessageResponse,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return ZboxMessageResponse, resp, err
}

func (c *ZboxClient) PostUserInfoAvatar(t *test.SystemTest, filePath, idToken, csrfToken, phoneNumber string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting user info avatar using 0box...")
	var ZboxMessageResponse *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/avatar")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxMessageResponse,
		FileName: "avatar",
		FilePath: filePath,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpFileUploadMethod)

	return ZboxMessageResponse, resp, err
}

func (c *ZboxClient) PostUserInfoBackgroundImage(t *test.SystemTest, filePath, idToken, csrfToken, phoneNumber string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting user info background using 0box...")
	var ZboxMessageResponse *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/userinfo/bgimg")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxMessageResponse,
		FileName: "background",
		FilePath: filePath,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpFileUploadMethod)

	return ZboxMessageResponse, resp, err
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
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
		},
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
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return zboxUsername, resp, err
}

func (c *ZboxClient) GetGraphWritePrice(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph write price using 0box...")
	var graphWritePrice model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/graph-write-price")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &graphWritePrice,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &graphWritePrice, resp, err
}

func (c *ZboxClient) GetShareInfo(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (model.ZboxShareInfoList, *resty.Response, error) {
	t.Logf("Getting share Info for [%v] using 0box...", phoneNumber)
	var ZboxShareInfoList model.ZboxShareInfoList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo/received")

	formData := map[string]string{
		"share_info_type": "public",
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxShareInfoList,
		QueryParams: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
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
	urlBuilder.SetPath("/v2/shareinfo")

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
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
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
	urlBuilder.SetPath("/v2/shareinfo")

	queryParams := map[string]string{
		"auth_ticket": authTicket,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &message,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)

	return message, resp, err
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
			"X-App-Client-ID":        "31f740fb12cf72464419a7e860591058a248b01e34b13cbf71d5a107b7bdc1e9",
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Timestamp":        "1618213324",
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZboxClient) CheckPhoneExists(t *test.SystemTest, csrfToken, phoneNumber string) (model.ZboxResourceExist, *resty.Response, error) {
	t.Logf("Checking if phone number [%v] exists using 0box...", phoneNumber)
	var zboxWalletExists model.ZboxResourceExist

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/phone/exist")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &zboxWalletExists,
		Headers: map[string]string{
			"X-App-User-ID": phoneNumber,
			"X-CSRF-TOKEN":  csrfToken,
			"X-APP-TYPE":    "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxWalletExists, resp, err
}

func (c *ZboxClient) CheckWalletExists(t *test.SystemTest, walletName, csrfToken, phoneNumber string) (model.ZboxResourceExist, *resty.Response, error) {
	t.Logf("Checking if wallet exists for [%v] using 0box...", phoneNumber)
	var zboxWalletExists model.ZboxResourceExist

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet/exist")
	formData := map[string]string{
		"wallet_name": walletName,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxWalletExists,
		FormData: formData,
		Headers: map[string]string{
			"X-App-User-ID": phoneNumber,
			"X-CSRF-TOKEN":  csrfToken,
			"X-APP-TYPE":    "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxWalletExists, resp, err
}

func (c *ZboxClient) CreateFCMToken(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*resty.Response, error) {
	t.Logf("Creating fcm token for [%v] using 0box...", phoneNumber)

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/fcmtoken")
	formData := map[string]string{
		"token":        idToken,
		"id_token":     idToken,
		"phone_number": phoneNumber,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Body: formData,
		Headers: map[string]string{
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-ID-TOKEN":   idToken,
			"X-App-User-ID":    phoneNumber,
			"X-CSRF-TOKEN":     csrfToken,
			"X-APP-TYPE":       "blimp",
			"X-App-Timestamp":  "1618213324",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return resp, err
}

func (c *ZboxClient) CreateNftCollectionId(t *test.SystemTest, idToken, csrfToken, phoneNumber, createdBy, collectionName, collectionId, authTicket, totalNfts, collectionType, allocationId, baseUrl, symbol string, pricePerPack, maxMints, currMints, batchSize int) (*model.ZboxNftCollection, *resty.Response, error) {
	t.Logf("Creating nft collection using 0box...")
	var ZboxNftCollection *model.ZboxNftCollection

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	formData := map[string]string{
		"created_by":      createdBy,
		"collection_name": collectionName,
		"collection_id":   collectionId,
		"auth_ticket":     authTicket,
		"total_nfts":      totalNfts,
		"collection_type": collectionType,
		"allocation_id":   allocationId,
		"base_url":        baseUrl,
		"symbol":          symbol,
		"price_per_pack":  strconv.Itoa(pricePerPack),
		"max_mints":       strconv.Itoa(maxMints),
		"curr_mints":      strconv.Itoa(currMints),
		"batch_size":      strconv.Itoa(batchSize),
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxNftCollection,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return ZboxNftCollection, resp, err
}

func (c *ZboxClient) PostNftCollection(t *test.SystemTest, idToken, csrfToken, phoneNumber, stage_nft_upload, nft_reference, collectionId, authTicket, owned_by, nft_activity, meta_data, allocationId, created_by, contract_address, token_id, token_standard, tx_hash string) (*model.ZboxNft, *resty.Response, error) {
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
		"auth_ticket":      authTicket,
		"owned_by":         owned_by,
		"nft_activity":     nft_activity,
		"meta_data":        meta_data,
		"allocation_id":    allocationId,
		"created_by":       created_by,
		"contract_address": contract_address,
		"token_id":         token_id,
		"token_standard":   token_standard,
		"tx_hash":          tx_hash,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxNft,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return ZboxNft, resp, err
}

func (c *ZboxClient) UpdateNftCollection(t *test.SystemTest, idToken, csrfToken, phoneNumber, createdBy, collectionName, collectionId, authTicket, totalNfts, collectionType, allocationId, baseUrl, symbol string, nftId, pricePerPack, maxMints, currMints, batchSize int) (*model.ZboxNft, *resty.Response, error) {
	t.Logf("Updating nft using 0box...")
	var ZboxNft *model.ZboxNft

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	formData := map[string]string{
		"created_by":      createdBy,
		"collection_name": collectionName,
		"collection_id":   collectionId,
		"auth_ticket":     authTicket,
		"total_nfts":      totalNfts,
		"collection_type": collectionType,
		"allocation_id":   allocationId,
		"base_url":        baseUrl,
		"symbol":          symbol,
		"price_per_pack":  strconv.Itoa(pricePerPack),
		"max_mints":       strconv.Itoa(maxMints),
		"curr_mints":      strconv.Itoa(currMints),
		"batch_size":      strconv.Itoa(batchSize),
	}

	queryParams := map[string]string{
		"id": strconv.Itoa(nftId),
	}
	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNft,
		FormData:    formData,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 201,
	}, HttpPUTMethod)

	return ZboxNft, resp, err
}

func (c *ZboxClient) GetAllNft(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxNftList, *resty.Response, error) {
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
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftList, resp, err
}

func (c *ZboxClient) GetAllNftByCollectionId(t *test.SystemTest, idToken, csrfToken, phoneNumber, collection_id string) (*model.ZboxNftListByCollection, *resty.Response, error) {
	t.Logf("Getting All nft using collection Id for 0box...")
	var ZboxNftList *model.ZboxNftListByCollection

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/bycollection")

	queryParams := map[string]string{
		"collection_id": collection_id,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNftList,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftList, resp, err
}

func (c *ZboxClient) GetAllNftByWalletId(t *test.SystemTest, idToken, csrfToken, phoneNumber, wallet_id string) (*model.ZboxNftListByWalletID, *resty.Response, error) {
	t.Logf("Getting All nft using wallet Id for 0box...")
	var ZboxNftList *model.ZboxNftListByWalletID

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/wallet")

	queryParams := map[string]string{
		"wallet_id": wallet_id,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNftList,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftList, resp, err
}

func (c *ZboxClient) UpdateFCMToken(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxFCMResponse, *resty.Response, error) {
	t.Logf("Updating fcm token for [%v] using 0box...", phoneNumber)
	var dest model.ZboxFCMResponse
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/fcmtoken")
	//todo: figure out which field can be updated
	formData := map[string]string{
		"fcm_token":             idToken,
		"fcm_token_device_type": "zorro",
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:  &dest,
		Body: formData,
		Headers: map[string]string{
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-ID-TOKEN":   idToken,
			"X-App-User-ID":    phoneNumber,
			"X-CSRF-TOKEN":     csrfToken,
			"X-APP-TYPE":       "blimp",
			"X-App-Timestamp":  "1618213326",
		},
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return &dest, resp, err
}

func (c *ZboxClient) DeleteFCMToken(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxFCMResponse, *resty.Response, error) {
	t.Logf("Deleting fcm token for [%v] using 0box...", phoneNumber)
	var dest model.ZboxFCMResponse
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/fcmtoken")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &dest,
		Headers: map[string]string{
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-ID-TOKEN":   idToken,
			"X-App-User-ID":    phoneNumber,
			"X-CSRF-TOKEN":     csrfToken,
			"X-APP-TYPE":       "blimp",
			"X-App-Timestamp":  "1618213426",
		},
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)

	return &dest, resp, err
}

func (c *ZboxClient) GetGraphTotalChallengePools(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph total challenge pools using 0box...")
	var graphTotalChallengePools model.ZboxGraphInt64Response
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/graph-total-challenge-pools")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &graphTotalChallengePools,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &graphTotalChallengePools, resp, err
}

func (c *ZboxClient) GetAllNftCollectionId(t *test.SystemTest, idToken, csrfToken, phoneNumber string) (*model.ZboxNftCollectionList, *resty.Response, error) {
	t.Logf("Getting All nft collection id using 0box...")
	var ZboxNftCollectionList *model.ZboxNftCollectionList
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collections")

	queryParams := map[string]string{}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNftCollectionList,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftCollectionList, resp, err
}

func (c *ZboxClient) GetNftCollectionById(t *test.SystemTest, idToken, csrfToken, phoneNumber, collection_id string) (*model.ZboxNftCollectionById, *resty.Response, error) {
	t.Logf("Getting All nft collection using collection Id for 0box...")
	var ZboxNftCollection *model.ZboxNftCollectionById
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	queryParams := map[string]string{
		"collection_id": collection_id,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:         &ZboxNftCollection,
		QueryParams: queryParams,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID,
			"X-App-Client-Key":       X_APP_CLIENT_KEY,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-APP-TYPE":             "blimp",
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftCollection, resp, err
}

func (c *ZboxClient) GetGraphAllocatedStorage(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph allocated storage using 0box...")
	var graphAllocatedStorage model.ZboxGraphInt64Response
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-allocated-storage")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &graphAllocatedStorage,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &graphAllocatedStorage, resp, err
}

func (c *ZboxClient) GetGraphUsedStorage(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph used storage using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-used-storage")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphTotalStaked(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph total staked using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-total-staked")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphTotalMinted(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph total minted using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-total-minted")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphTotalLocked(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph total locked using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-total-locked")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphChallenges(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphChallengesResponse, *resty.Response, error) {
	t.Logf("Getting graph challenges using 0box...")
	var data model.ZboxGraphChallengesResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-challenges")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphTokenSupply(t *test.SystemTest, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph token supply using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-token-supply")
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetTotalMinted(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting total minted using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-minted")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetTotalBlobberCapacity(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting total blobber capacity using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-blobber-capacity")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetAverageWritePrice(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting average write price using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/average-write-price")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetTotalStaked(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting total staked using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-staked")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetTotalChallenges(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting total challenges using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-total-challenges")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetSuccessfulChallenges(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting Successful Challenges using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-successful-challenges")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetTotalAllocatedStorage(t *test.SystemTest) (*model.ZboxTotalInt64Response, *resty.Response, error) {
	t.Logf("Getting Allocated Storage using 0box...")
	var data model.ZboxTotalInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/total-allocated-storage")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberChallengesPassed(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber challenges passed using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-challenges-passed")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberChallengesCompleted(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber challenges completed using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-challenges-completed")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberChallengesOpen(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber challenges open using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-challenges-open")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberInactiveRounds(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber inactive rounds using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-inactive-rounds")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberWritePrice(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber write price using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-write-price")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberCapacity(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber capacity using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-capacity")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberAllocated(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber allocated using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-allocated")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetGraphBlobberSavedData(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber saved data using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-saved-data")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

// GetGraphBlobberReadData returns the blobber read price data
func (c *ZboxClient) GetGraphBlobberReadData(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber read data using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-read-data")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

// GetGraphBlobberOffersTotal returns the blobber offers total
func (c *ZboxClient) GetGraphBlobberOffersTotal(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber offers total using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-offers-total")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

// GetGraphBlobberStakeTotal
func (c *ZboxClient) GetGraphBlobberTotalStake(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber total stake using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-total-stake")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

// GetGraphBlobberStakeTotal
func (c *ZboxClient) GetGraphBlobberTotalRewards(t *test.SystemTest, blobberId string, req *model.ZboxGraphRequest) (*model.ZboxGraphInt64Response, *resty.Response, error) {
	t.Logf("Getting graph blobber total rewards using 0box...")
	var data model.ZboxGraphInt64Response

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")

	urlBuilder.SetPath("/v2/graph-blobber-total-rewards")
	urlBuilder.queries.Set("id", blobberId)
	urlBuilder.queries.Set("from", req.From)
	urlBuilder.queries.Set("to", req.To)
	urlBuilder.queries.Set("data-points", req.DataPoints)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &data,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &data, resp, err
}

func (c *ZboxClient) GetReferralCode(t *test.SystemTest, csrfToken, idToken, phoneNumber string) (model.ReferralCodeOfUser, *resty.Response, error) {
	t.Log("Getting referral code...")
	var ReferralCodeOfUser model.ReferralCodeOfUser

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/code/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &ReferralCodeOfUser,
		Headers: map[string]string{
			"X-App-User-ID":    phoneNumber,
			"X-APP-TYPE":       "blimp",
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-Timestamp":  "1618213324",
			"X-App-ID-TOKEN":   idToken,
			"X-CSRF-TOKEN":     csrfToken,
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralCodeOfUser, resp, err
}

func (c *ZboxClient) GetReferralCount(t *test.SystemTest, csrfToken, idToken, phoneNumber string) (model.ReferralCountOfUser, *resty.Response, error) {
	t.Log("Getting referral count...")
	var ReferralCountOfUser model.ReferralCountOfUser

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/count/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &ReferralCountOfUser,
		Headers: map[string]string{
			"X-App-User-ID":    phoneNumber,
			"X-APP-TYPE":       "blimp",
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-Timestamp":  "1618213324",
			"X-App-ID-TOKEN":   idToken,
			"X-CSRF-TOKEN":     csrfToken,
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralCountOfUser, resp, err
}

func (c *ZboxClient) GetLeaderBoard(t *test.SystemTest, csrfToken, idToken, phoneNumber string) ([]model.TopReferrer, *resty.Response, error) {
	t.Logf("Checking if wallet exists for [%v] using 0box...", phoneNumber)
	var ReferralLeaderBoard []model.TopReferrer

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/topusers/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &ReferralLeaderBoard,
		Headers: map[string]string{
			"X-App-User-ID":    phoneNumber,
			"X-APP-TYPE":       "blimp",
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-Timestamp":  "1618213324",
			"X-App-ID-TOKEN":   idToken,
			"X-CSRF-TOKEN":     csrfToken,
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralLeaderBoard, resp, err
}

func (c *ZboxClient) GetReferralRank(t *test.SystemTest, csrfToken, idToken, phoneNumber string) (model.ReferralRankOfUser, *resty.Response, error) {
	t.Log("Getting referral rank...")
	var ReferralRankOfUser model.ReferralRankOfUser

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/userrank/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &ReferralRankOfUser,
		Headers: map[string]string{
			"X-App-User-ID":    phoneNumber,
			"X-APP-TYPE":       "blimp",
			"X-App-Client-ID":  X_APP_CLIENT_ID,
			"X-App-Client-Key": X_APP_CLIENT_KEY,
			"X-App-Timestamp":  "1618213324",
			"X-App-ID-TOKEN":   idToken,
			"X-CSRF-TOKEN":     csrfToken,
		},
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralRankOfUser, resp, err
}

func (c *ZboxClient) PostOwnerWithReferralCode(t *test.SystemTest, idToken, csrfToken, phoneNumber, appType, userName string) (*model.ZboxOwner, *resty.Response, error) {
	t.Logf("Posting owner using 0box...")
	var zboxOwner *model.ZboxOwner

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/owner")

	formData := map[string]string{
		"username": userName,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxOwner,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID_R,
			"X-App-Client-Key":       X_APP_CLIENT_KEY_R,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE_R,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Type":             appType,
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return zboxOwner, resp, err
}

func (c *ZboxClient) PostWalletWithReferralCode(t *test.SystemTest, mnemonic, walletName, walletDescription, idToken, csrfToken, phoneNumber, appType, userName, refCode string) (*model.ZboxWallet, *resty.Response, error) {
	t.Logf("Posting wallet with referral code using 0box...")
	var zboxWallet *model.ZboxWallet

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/wallet")

	formData := map[string]string{
		"mnemonic":    mnemonic,
		"name":        walletName,
		"description": walletDescription,
		"refcode":     refCode,
		"user_name":   userName,
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &zboxWallet,
		FormData: formData,
		Headers: map[string]string{
			"X-App-Client-ID":        X_APP_CLIENT_ID_R,
			"X-App-Client-Key":       X_APP_CLIENT_KEY_R,
			"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE_R,
			"X-App-Timestamp":        "1618213324",
			"X-App-ID-TOKEN":         idToken,
			"X-App-User-ID":          phoneNumber,
			"X-CSRF-TOKEN":           csrfToken,
			"X-App-Type":             appType,
		},
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return zboxWallet, resp, err
}
