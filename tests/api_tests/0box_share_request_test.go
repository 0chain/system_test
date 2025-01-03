package api_tests

import (
	"strconv"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestShareRequest(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// create receivedUserHeaders, receivedUserHeaders2, requestedUserHeaders, requestedUserHeaders2
	receivedUserHeaders := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_A,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_A,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_A,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_A,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	receivedClientID := client.X_APP_CLIENT_ID_A

	receivedUserHeaders2 := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_B,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_B,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_B,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_B,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	receivedClientID2 := client.X_APP_CLIENT_ID_B

	requestedUserHeaders := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	requestedClientID := client.X_APP_CLIENT_ID

	requestedUserHeaders2 := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_R,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_R,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_R,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_R,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	requestedClientID2 := client.X_APP_CLIENT_ID_R

	// create wallet for receivedClientID, requestedClientID, requestedClientID2
	err := Create0boxTestWallet(t, receivedUserHeaders)
	require.NoError(t, err)
	err = Create0boxTestWallet(t, receivedUserHeaders2)
	require.NoError(t, err)
	err = Create0boxTestWallet(t, requestedUserHeaders)
	require.NoError(t, err)
	err = Create0boxTestWallet(t, requestedUserHeaders2)
	require.NoError(t, err)

	// data
	shareRequestData1 := map[string]string{
		"authticket":  "authticket1",
		"message":     "provide access!",
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	}
	shareRequestData2 := map[string]string{
		"authticket":  "authticket2",
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0b",
	}
	shareRequestData3 := map[string]string{
		"authticket":  "authticket3",
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	}
	shareRequestData4 := map[string]string{
		"authticket":  "authticket4",
		"owner_id":    receivedClientID2,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0c",
	}

	// api response
	var (
		reqId1, reqId2, reqId3, reqId4, reReqId int64
	)

	t.RunSequentially("verify empty requests", func(t *test.SystemTest) {
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{})
		require.NoError(t, err, "error empty received shareReq")
		require.Empty(t, dataResp.Data, "data list should be empty")

		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{})
		require.NoError(t, err, "error empty requested shareReq")
		require.Empty(t, dataResp.Data, "data list should be empty")
	})

	t.RunSequentially("verify create requests", func(t *test.SystemTest) {
		// create request by requestedClientID to receivedClientID
		dataResp, _, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.NoError(t, err, "error creating shareRequestData1")
		reqId1 = dataResp.Data

		// re-request with same data
		_, restyResp, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.Error(t, err, "re-request to already created request should fail")
		require.Equal(t, "400", restyResp.Status())
		require.Contains(t, restyResp.String(), "request_already_created")

		// create request by requestedClientID to receivedClientID for different file
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData2)
		require.NoError(t, err, "error creating shareRequestData2")
		reqId2 = dataResp.Data

		// create request by requestedClientID2 to receivedClientID
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUserHeaders2, shareRequestData3)
		require.NoError(t, err, "error creating shareRequestData3")
		reqId3 = dataResp.Data

		// create request by requestedClientID2 to receivedClientID2
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUserHeaders2, shareRequestData4)
		require.NoError(t, err, "error creating shareRequestData4")
		reqId4 = dataResp.Data
	})

	t.RunSequentially("verify get all requests", func(t *test.SystemTest) {
		// verify get all received requests
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, len(dataResp.Data), 3)
		require.Equal(t, []int64{reqId3, reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID, dataResp.Data[2].ID})

		// verify get all received requests by requestedClientID
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":       "true",
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, len(dataResp.Data), 2)
		require.Equal(t, []int64{reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all received requests by requestedClientID2
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":       "true",
			"client_id": requestedClientID2,
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reqId3}, []int64{dataResp.Data[0].ID})

		// verify get all received requests by requestedClientID and LookupHash
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":         "true",
			"client_id":   requestedClientID,
			"lookup_hash": shareRequestData1["lookup_hash"],
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get all share requested by requestedClientID
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, len(dataResp.Data), 2)
		require.Equal(t, []int64{reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all share requested by requestedClientID with LookupHash filter
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{
			"all":         "true",
			"lookup_hash": shareRequestData1["lookup_hash"],
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get all share requested by requestedClientID2
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders2, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, len(dataResp.Data), 2)
		require.Equal(t, []int64{reqId4, reqId3}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all share requested by requestedClientID2 with OwnerID filter
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders2, map[string]string{
			"all":      "true",
			"owner_id": receivedClientID2,
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reqId4}, []int64{dataResp.Data[0].ID})
	})

	t.RunSequentially("update requests", func(t *test.SystemTest) {
		// approve reqId1
		dataResp, _, err := zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId1, 10),
			"status": "1",
		})
		require.NoError(t, err, "error updating reqId1")
		require.Equal(t, 1, dataResp.Data.Status, "reqId1 should be approved")

		// approve or decline reqId1 again should fail
		dataResp, restyResp, err := zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId1, 10),
			"status": "2",
		})
		require.Error(t, err, "update to already approved request should fail")
		require.Equal(t, "400", restyResp.Status())
		require.Contains(t, restyResp.String(), "status is in final state")

		// approve reqId2 by receivedClientID2 should fail authorization
		dataResp, restyResp, err = zboxClient.UpdateShareReq(t, receivedUserHeaders2, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "1",
		})
		require.Error(t, err, "approve reqId2 by receivedClientID2 should fail authorization")
		require.Equal(t, "400", restyResp.Status())
		require.Contains(t, restyResp.String(), "unauthorized:")

		// status other than 1 or 2 should fail
		dataResp, restyResp, err = zboxClient.UpdateShareReq(t, receivedUserHeaders2, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "0",
		})
		require.Error(t, err, "status other than 1 or 2 should fail")
		require.Equal(t, "400", restyResp.Status())
		require.Contains(t, restyResp.String(), "invalid status")

		// decline reqId2
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "2",
		})
		require.NoError(t, err, "error updating reqId2")
		require.Equal(t, 2, dataResp.Data.Status, "reqId2 should be declined")

		// decline reqId3
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId3, 10),
			"status": "2",
		})
		require.NoError(t, err, "error updating reqId3")
		require.Equal(t, 2, dataResp.Data.Status, "reqId3 should be declined")

		// approve reqId4
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUserHeaders2, map[string]string{
			"id":     strconv.FormatInt(reqId4, 10),
			"status": "1",
		})
		require.NoError(t, err, "error updating reqId4")
		require.Equal(t, 1, dataResp.Data.Status, "reqId4 should be declined")
	})

	t.RunSequentially("verify re-request for approved and declined shareReq", func(t *test.SystemTest) {
		// re-request for already approved request should fail
		_, restyResp, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.Error(t, err, "re-request for already approved request should fail")
		require.Equal(t, "400", restyResp.Status())
		require.Contains(t, restyResp.String(), "request_already_approved")

		// re-request for declined request should be allowed
		dataResp, _, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData2)
		require.NoError(t, err, "re-request for declined request should be allowed")
		reReqId = dataResp.Data
	})

	t.RunSequentially("verify get latest requests", func(t *test.SystemTest) {
		// verify get latest approved request
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"status":    "1",
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting latest approved shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get latest request
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting latest approved shareReq")
		require.Len(t, len(dataResp.Data), 1)
		require.Equal(t, []int64{reReqId}, []int64{dataResp.Data[0].ID})
		require.Equal(t, 0, dataResp.Data[0].Status)
	})

}
