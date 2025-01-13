package client

import (
	"fmt"
	"strconv"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

const (
	X_APP_USER_ID            = "test_user_id"
	X_APP_CLIENT_ID          = "caae5a9d48b1a0cd01a5da982807d7ad6fcc8a8367b79c6adbe48e7c632544f2"
	X_APP_CLIENT_KEY         = "91a3a29f4c05b82f2a83f9d4b405976637a4a29f11b1918de30fc319ab87db191b195d9f3e6eecf588e1b83d195931d12760f303c3d1845144f07617022faa8f"
	X_APP_CLIENT_SIGNATURE   = "6de25e5a202614216610558ec68346a9aca97165e29a12cc047fbeb8c696d420"
	X_APP_USER_ID_A          = "test_user_id_alternative"
	X_APP_CLIENT_ID_A        = "2e16f28602959e23e0f5b153e298fc5dbfb02dcb089cfa74bd46f158c2f02ab7"
	X_APP_CLIENT_KEY_A       = "f615cfbb9154c99bf6e1d87e26397e987ee345c2e98c7a652326a3744f15ea100e22506fccf75515d9379f3275386f4401e2dbd1aafc4ef8f18af58a8c68fc02"
	X_APP_CLIENT_SIGNATURE_A = "db4bf04ab302a4dde154804334d0eeb512d875585a4092c7776cf3505ba9e793"
	X_APP_USER_ID_R          = "test_user_id_referred_user"
	X_APP_CLIENT_ID_R        = "bcf5f517be521e0ffdb22d1fc26a35abdec8556bcb9ed075244a358df7337cd0"
	X_APP_CLIENT_KEY_R       = "de9351bbf460c761ea979764831759369ced3c6de38b856e7921eee9cc034323cce23c562c74a0629867fbea33d5009b9f473bc3b766871b65e7cf864ba4301d"
	X_APP_CLIENT_SIGNATURE_R = "43fa947257ae0b5da8b073f0efeaa2570c4faf66fcabd93407e3597bc34e440b"
	X_APP_USER_ID_B          = "test_user_id_B"
	X_APP_CLIENT_ID_B        = "fc6ed0246c8bb4f7251acf521c81d7fdb3f042a7bb2234a032723c9bba8dda9b"
	X_APP_CLIENT_KEY_B       = "92843dc9dafc041e88b778356bf39533606911828d50f420fb24e0cc2dcc4b06a8ed6dc0f083c988bc8cc64c7525f943dff8d1ece4955e456c450d34faf8da12"
	X_APP_CLIENT_SIGNATURE_B = "11648e1f79b90419166c13f09d9bdf15e32105591c1f94d461aa82c569480f17"
	X_APP_ID_TOKEN           = "test_firebase_token"
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

func (c *ZboxClient) NewZboxPublicHeaders(appType string) map[string]string {
	zboxHeaders := map[string]string{
		"X-App-User-ID": X_APP_USER_ID,
		"X-APP-TYPE":    appType,
	}
	return zboxHeaders
}

func (c *ZboxClient) NewZboxCSRFHeaders(appType string) map[string]string {
	zboxHeaders := map[string]string{
		"X-App-User-ID": X_APP_USER_ID,
		"X-CSRF-TOKEN":  X_APP_CSRF,
		"X-APP-TYPE":    appType,
	}
	return zboxHeaders
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
		"X-App-User-ID":          X_APP_USER_ID_R,
		"X-CSRF-TOKEN":           X_APP_CSRF,
		"X-App-Client-Signature": X_APP_CLIENT_SIGNATURE_R,
		"X-APP-TYPE":             appType,
	}
	return zboxHeaders
}

func (c *ZboxClient) VerifyOtpDetails(t *test.SystemTest, headers, verifyOtpInput map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("verify otp and creating owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxMessageRes *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/twilio/phone/verify/signup")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxMessageRes,
		FormData:           verifyOtpInput,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)
	return zboxMessageRes, resp, err
}

func (c *ZboxClient) UpdateOwner(t *test.SystemTest, headers, owner map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("updating owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.ZboxMessageResponse

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

func (c *ZboxClient) DeleteOwner(t *test.SystemTest, headers map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("deleting owner for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.ZboxMessageResponse

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

func (c *ZboxClient) UpdateWallet(t *test.SystemTest, headers, wallet map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("updating wallet for userID [%v] using 0box...", headers["X-App-User-ID"])
	var message *model.ZboxMessageResponse

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

func (c *ZboxClient) CreateAllocation(t *test.SystemTest, headers, allocation map[string]string) (*model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocation["id"])
	var zboxAllocation *model.ZboxAllocation

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

func (c *ZboxClient) UpdateAllocation(t *test.SystemTest, headers, allocation map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocation["id"])
	var updateResponse *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/allocation")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &updateResponse,
		FormData:           allocation,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return updateResponse, resp, err
}

func (c *ZboxClient) GetAllocation(t *test.SystemTest, headers map[string]string, allocationID string) (*model.ZboxAllocation, *resty.Response, error) {
	t.Logf("Getting allocation for  allocationId [%v] using 0box...", allocationID)
	var zboxAllocation *model.ZboxAllocation

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

func (c *ZboxClient) GetReferralCode(t *test.SystemTest, headers map[string]string) (*model.ReferralCodeOfUser, *resty.Response, error) {
	t.Log("Getting referral code...")
	var referralCodeOfUser *model.ReferralCodeOfUser

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/code/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &referralCodeOfUser,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return referralCodeOfUser, resp, err
}

func (c *ZboxClient) GetReferralCount(t *test.SystemTest, headers map[string]string) (*model.ReferralCount, *resty.Response, error) {
	t.Log("Getting referral count...")
	var ReferralCountOfUser *model.ReferralCount

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/count/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ReferralCountOfUser,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralCountOfUser, resp, err
}

func (c *ZboxClient) GetLeaderBoard(t *test.SystemTest, headers map[string]string) (*model.TopReferrerResponse, *resty.Response, error) {
	t.Logf("getting referral leader board")
	var topReferrers *model.TopReferrerResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/topusers/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &topReferrers,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return topReferrers, resp, err
}

func (c *ZboxClient) GetReferralRank(t *test.SystemTest, headers map[string]string) (*model.ReferralRankOfUser, *resty.Response, error) {
	t.Log("Getting referral rank...")
	var ReferralRankOfUser *model.ReferralRankOfUser

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/referral/userrank/")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ReferralRankOfUser,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ReferralRankOfUser, resp, err
}

func (c *ZboxClient) GetDexState(t *test.SystemTest, headers map[string]string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst:                &dexState,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return dexState, resp, err
}

func (c *ZboxClient) CreateDexState(t *test.SystemTest, headers, data map[string]string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	formData := data

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst:                &dexState,
		FormData:           formData,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPOSTMethod)

	return dexState, resp, err
}

func (c *ZboxClient) UpdateDexState(t *test.SystemTest, headers, data map[string]string) (*model.DexState, *resty.Response, error) {
	t.Log("Posting Dex state using 0box...")
	var dexState *model.DexState

	urBuilder := NewURLBuilder()
	err := urBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urBuilder.SetPath("/v2/dex/state")

	formData := data

	resp, err := c.executeForServiceProvider(t, urBuilder.String(), model.ExecutionRequest{
		Dst:                &dexState,
		FormData:           formData,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return dexState, resp, err
}

func (c *ZboxClient) CheckFundingStatus(t *test.SystemTest, headers map[string]string, fundingId string) (*model.ZboxFundingResponse, *resty.Response, error) {
	t.Logf("Checking status of funding using funding id")
	var zboxFundingResponse *model.ZboxFundingResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/zbox/fund")

	url := fmt.Sprintf("%s/%s", urlBuilder.String(), fundingId)
	resp, err := c.executeForServiceProvider(t, url, model.ExecutionRequest{
		Dst:                &zboxFundingResponse,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return zboxFundingResponse, resp, err
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

func (c *ZboxClient) GetShareInfoShared(t *test.SystemTest, headers map[string]string) (*model.ZboxMessageDataShareinfoResponse, *resty.Response, error) {
	t.Logf("Getting share Info for [%v] using 0box...", headers["X-App-User-ID"])
	var ZboxShareInfoList *model.ZboxMessageDataShareinfoResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo/shared")

	formData := map[string]string{
		"share_info_type": "public",
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxShareInfoList,
		QueryParams:        formData,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxShareInfoList, resp, err
}

func (c *ZboxClient) GetShareInfoReceived(t *test.SystemTest, headers map[string]string) (*model.ZboxMessageDataShareinfoResponse, *resty.Response, error) {
	t.Logf("Getting share Info for [%v] using 0box...", headers["X-App-User-ID"])
	var ZboxShareInfoList *model.ZboxMessageDataShareinfoResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo/received")

	formData := map[string]string{
		"share_info_type": "public",
	}

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxShareInfoList,
		QueryParams:        formData,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxShareInfoList, resp, err
}

func (c *ZboxClient) GetShareInfoReceivedWithParams(t *test.SystemTest, headers map[string]string, queryParams map[string]string) (*model.ZboxMessageDataShareinfoResponse, *resty.Response, error) {
	t.Logf("Getting share Info for [%v] using 0box...", headers["X-App-User-ID"])
	var ZboxShareInfoList *model.ZboxMessageDataShareinfoResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo/received")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxShareInfoList,
		QueryParams:        queryParams,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxShareInfoList, resp, err
}

func (c *ZboxClient) CreateShareInfo(t *test.SystemTest, headers, shareinfoData map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting ShareInfo using 0box...")
	var message *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &message,
		FormData:           shareinfoData,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)
	return message, resp, err
}

func (c *ZboxClient) DeleteShareinfo(t *test.SystemTest, headers map[string]string, authTicket string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting ShareInfo using 0box...")
	var message *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/shareinfo")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &message,
		QueryParams: map[string]string{
			"auth_ticket": authTicket,
		},
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)
	return message, resp, err
}

func (c *ZboxClient) CreateShareRequest(t *test.SystemTest, headers, shareRequestData map[string]string) (*model.ZboxMessageDataResponse[int64], *resty.Response, error) {
	t.Logf("Posting ShareRequest using 0box...")
	var out model.ZboxMessageDataResponse[int64]

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/sharereq")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &out,
		Body:               shareRequestData,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)
	return &out, resp, err
}

func (c *ZboxClient) GetReceivedShareReq(t *test.SystemTest, headers map[string]string, queryParams map[string]string) (*model.ZboxMessageDataResponse[[]model.ZboxShareRequest], *resty.Response, error) {
	t.Logf("Getting received share requests for [%v] using 0box...", headers["X-App-User-ID"])
	var out model.ZboxMessageDataResponse[[]model.ZboxShareRequest]

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/sharereq/received")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &out,
		QueryParams:        queryParams,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &out, resp, err
}

func (c *ZboxClient) GetRequestedShareReq(t *test.SystemTest, headers map[string]string, queryParams map[string]string) (*model.ZboxMessageDataResponse[[]model.ZboxShareRequest], *resty.Response, error) {
	t.Logf("Getting requested share requests for [%v] using 0box...", headers["X-App-User-ID"])
	var out model.ZboxMessageDataResponse[[]model.ZboxShareRequest]

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/sharereq/requested")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &out,
		QueryParams:        queryParams,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &out, resp, err
}

func (c *ZboxClient) UpdateShareReq(t *test.SystemTest, headers, updateShareReqData map[string]string) (*model.ZboxMessageDataResponse[model.ZboxShareRequest], *resty.Response, error) {
	t.Logf("Updating ShareRequest using 0box...")
	var out model.ZboxMessageDataResponse[model.ZboxShareRequest]

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/sharereq")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &out,
		FormData:           updateShareReqData,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPUTMethod)
	return &out, resp, err
}

func (c *ZboxClient) DeleteShareReq(t *test.SystemTest, headers, queryParams map[string]string) (*model.ZboxMessageDataResponse[int64], *resty.Response, error) {
	t.Logf("Deleting ShareRequest using 0box...")
	var out model.ZboxMessageDataResponse[int64]

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/sharereq")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &out,
		QueryParams:        queryParams,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpDELETEMethod)
	return &out, resp, err
}

func (c *ZboxClient) CreateNftCollection(t *test.SystemTest, headers, nfCollectionData map[string]string) (*model.ZboxNftCollection, *resty.Response, error) {
	t.Logf("Creating nft collection using 0box...")
	var ZboxNftCollection *model.ZboxNftCollection

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxNftCollection,
		FormData:           nfCollectionData,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return ZboxNftCollection, resp, err
}

func (c *ZboxClient) UpdateNftCollection(t *test.SystemTest, headers, nftCollectionData map[string]string) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Creating nft collection using 0box...")
	var ZboxNftCollection *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collection")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxNftCollection,
		FormData:           nftCollectionData,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpPUTMethod)

	return ZboxNftCollection, resp, err
}

func (c *ZboxClient) GetNftCollections(t *test.SystemTest, headers map[string]string) (*model.ZboxNftCollectionList, *resty.Response, error) {
	t.Logf("Creating nft collection using 0box...")
	var ZboxNftCollections *model.ZboxNftCollectionList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/collections")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxNftCollections,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftCollections, resp, err
}

func (c *ZboxClient) CreateNft(t *test.SystemTest, headers, nftData map[string]string) (*model.ZboxNft, *resty.Response, error) {
	t.Logf("Posting nft using 0box...")
	var ZboxNft *model.ZboxNft

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:      &ZboxNft,
		FormData: nftData,
		Headers:  headers},
		HttpPOSTMethod)

	return ZboxNft, resp, err
}

func (c *ZboxClient) UpdateNft(t *test.SystemTest, headers, nftData map[string]string, id int64) (*model.ZboxMessageResponse, *resty.Response, error) {
	t.Logf("Posting nft using 0box...")
	var ZboxNft *model.ZboxMessageResponse

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst: &ZboxNft,
		QueryParams: map[string]string{
			"id": strconv.FormatInt(id, 10),
		},
		FormData: nftData,
		Headers:  headers},
		HttpPUTMethod)

	return ZboxNft, resp, err
}

func (c *ZboxClient) GetAllNfts(t *test.SystemTest, headers map[string]string) (*model.ZboxNftList, *resty.Response, error) {
	t.Logf("Getting All nft using 0box...")
	var ZboxNftList *model.ZboxNftList

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/nft/all")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &ZboxNftList,
		Headers:            headers,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return ZboxNftList, resp, err
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

func (c *ZboxClient) CreateJwtToken(t *test.SystemTest, headers map[string]string) (*model.ZboxJwtToken, *resty.Response, error) {
	t.Logf("creating jwt token for userID [%v] using 0box...", headers["X-App-User-ID"])
	var zboxJwtToken *model.ZboxJwtToken

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/jwt/token")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxJwtToken,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPOSTMethod)

	return zboxJwtToken, resp, err
}

func (c *ZboxClient) RefreshJwtToken(t *test.SystemTest, token string, headers map[string]string) (*model.ZboxJwtToken, *resty.Response, error) {
	t.Logf("refreshing jwt token for userID [%v] and token [%v] using 0box...", headers["X-App-User-ID"], token)
	var zboxJwtToken *model.ZboxJwtToken

	headers["X-Jwt-Token"] = token

	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/jwt/token")

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &zboxJwtToken,
		Headers:            headers,
		RequiredStatusCode: 201,
	}, HttpPUTMethod)

	return zboxJwtToken, resp, err
}

func (c *ZboxClient) GetTransactionsList(t *test.SystemTest, pitId string) (*model.ZboxTransactionsDataResponse, *resty.Response, error) {
	t.Logf("Getting transactions data with pitid using 0box...")
	var txnData model.ZboxTransactionsDataResponse
	urlBuilder := NewURLBuilder()
	err := urlBuilder.MustShiftParse(c.zboxEntrypoint)
	require.NoError(t, err, "URL parse error")
	urlBuilder.SetPath("/v2/transactions")
	urlBuilder.queries.Set("pit_id", pitId)

	resp, err := c.executeForServiceProvider(t, urlBuilder.String(), model.ExecutionRequest{
		Dst:                &txnData,
		RequiredStatusCode: 200,
	}, HttpGETMethod)

	return &txnData, resp, err
}
