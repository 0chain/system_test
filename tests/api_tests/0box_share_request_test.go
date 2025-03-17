package api_tests

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func TestShareRequest(testSetup *testing.T) {
	w := test.NewSystemTest(testSetup)

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
	receivedOwnerInfo := map[string]string{
		"username":       "test_owner_received",
		"email":          "test_email_received",
		"phone_number":   "+919876543211",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID_A,
	}
	receivedClientID := client.X_APP_CLIENT_ID_A

	receivedUser2Headers := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_B,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_B,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_B,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_B,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	receivedOwner2Info := map[string]string{
		"username":       "test_owner2_received",
		"email":          "test_email2_received",
		"phone_number":   "+919876543212",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID_B,
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
	requestedOwnerInfo := map[string]string{
		"username":       "test_owner_requested",
		"email":          "test_email_requested",
		"phone_number":   "+919876543213",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID,
	}
	requestedClientID := client.X_APP_CLIENT_ID

	requestedUser2Headers := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_R,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_R,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_R,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_R,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	requestedOwner2Info := map[string]string{
		"username":       "test_owner2_requested",
		"email":          "test_email2_requested",
		"phone_number":   "+919876543214",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID_R,
	}
	requestedClientID2 := client.X_APP_CLIENT_ID_R

	prevSharedUserHeaders := map[string]string{
		"X-App-Client-ID":        client.X_APP_CLIENT_ID_C,
		"X-App-Client-Key":       client.X_APP_CLIENT_KEY_C,
		"X-App-Timestamp":        client.X_APP_TIMESTAMP,
		"X-App-ID-TOKEN":         client.X_APP_ID_TOKEN,
		"X-App-User-ID":          client.X_APP_USER_ID_C,
		"X-CSRF-TOKEN":           client.X_APP_CSRF,
		"X-App-Client-Signature": client.X_APP_CLIENT_SIGNATURE_C,
		"X-APP-TYPE":             client.X_APP_VULT,
	}
	prevUserInfo := map[string]string{
		"username":       "random_user",
		"email":          "random_email",
		"phone_number":   "+919876543215",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID_C,
	}
	prevSharedClientID := client.X_APP_CLIENT_ID_C

	// create wallet for receivedClientID, requestedClientID, requestedClientID2
	Teardown(w, receivedUserHeaders)
	Teardown(w, receivedUser2Headers)
	Teardown(w, requestedUserHeaders)
	Teardown(w, requestedUser2Headers)
	Teardown(w, prevSharedUserHeaders)
	err := Create0boxTestWalletCustom(w, receivedUserHeaders, receivedOwnerInfo, NewTestWallet())
	require.NoError(w, err)
	err = Create0boxTestWalletCustom(w, receivedUser2Headers, receivedOwner2Info, NewTestWallet())
	require.NoError(w, err)
	err = Create0boxTestWalletCustom(w, requestedUserHeaders, requestedOwnerInfo, NewTestWallet())
	require.NoError(w, err)
	err = Create0boxTestWalletCustom(w, requestedUser2Headers, requestedOwner2Info, NewTestWallet())
	require.NoError(w, err)
	err = Create0boxTestWalletCustom(w, prevSharedUserHeaders, prevUserInfo, NewTestWallet())
	require.NoError(w, err)

	// reqId1: requestedClientID --> receivedClientID (file1)
	// reqId2: requestedClientID --> receivedClientID (file2)
	// reqId3: requestedClientID2 --> receivedClientID 
	// reqId4: requestedClientID2 --> receivedClientID2


	// data
	authTicket1, err := convertToAuthTicket(map[string]string{
		"client_id":      prevSharedClientID,
		"owner_id":       receivedClientID,
		"file_name":      "file1.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	})
	require.NoError(w, err, "error creating authTicket1")
	shareRequestData1 := map[string]string{
		"auth_ticket": authTicket1,
		"message":     "provide access!",
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	}
	approvalAuthTicket1, err := convertToAuthTicket(map[string]string{
		"client_id":      requestedClientID,
		"owner_id":       receivedClientID,
		"file_name":      "file1.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	})
	require.NoError(w, err, "error creating approvalAuthTicket1")
	authTicket2, err := convertToAuthTicket(map[string]string{
		"client_id":      prevSharedClientID,
		"owner_id":       receivedClientID,
		"file_name":      "file2.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0b",
	})
	require.NoError(w, err, "error creating authTicket2")
	shareRequestData2 := map[string]string{
		"auth_ticket": authTicket2,
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0b",
	}
	authTicket3, err := convertToAuthTicket(map[string]string{
		"client_id":      prevSharedClientID,
		"owner_id":       receivedClientID,
		"file_name":      "file3.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	})
	require.NoError(w, err, "error creating authTicket3")
	shareRequestData3 := map[string]string{
		"auth_ticket": authTicket3,
		"owner_id":    receivedClientID,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0a",
	}
	authTicket4, err := convertToAuthTicket(map[string]string{
		"client_id":      prevSharedClientID,
		"owner_id":       receivedClientID2,
		"file_name":      "file4.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0c",
	})
	require.NoError(w, err, "error creating authTicket3")
	approvalAuthTicket4, err := convertToAuthTicket(map[string]string{
		"client_id":      prevSharedClientID,
		"owner_id":       receivedClientID2,
		"file_name":      "file4.png",
		"file_path_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0c",
	})
	require.NoError(w, err, "error creating approvalAuthTicket4")
	
	shareRequestData4 := map[string]string{
		"auth_ticket": authTicket4,
		"owner_id":    receivedClientID2,
		"lookup_hash": "16a55237cfd477cca9f87cb51d14b1e64c6e87592429e9e471470ad177ddbe0c",
	}
	randomLink := "randomLink"

	// api response
	var (
		reqId1, reqId2, reqId3, reqId4, reReqId int64
		toDeleteIds                             []int64
	)

	w.Cleanup(func() {
		for _, deleteId := range toDeleteIds {
			dataResp, restyResp, err := zboxClient.DeleteShareReq(w, requestedUserHeaders, map[string]string{
				"id": strconv.FormatInt(deleteId, 10),
			})
			if err != nil {
				w.Errorf("error deleting ID %v : %v ", deleteId, restyResp.String())
				continue
			}
			w.Logf("deleted ID = %v ; count = %v", deleteId, dataResp.Data)
		}

		Teardown(w, receivedUserHeaders)
		Teardown(w, receivedUser2Headers)
		Teardown(w, requestedUserHeaders)
		Teardown(w, requestedUser2Headers)
	})

	w.RunSequentially("verify empty requests", func(t *test.SystemTest) {
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{})
		require.NoError(t, err, "error empty received shareReq")
		require.Empty(t, dataResp.Data, "data list should be empty")

		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{})
		require.NoError(t, err, "error empty requested shareReq")
		require.Empty(t, dataResp.Data, "data list should be empty")
	})

	w.RunSequentially("verify create requests", func(t *test.SystemTest) {
		// create request by requestedClientID to receivedClientID
		dataResp, _, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.NoError(t, err, "error creating shareRequestData1")
		reqId1 = dataResp.Data
		toDeleteIds = append(toDeleteIds, reqId1)

		// re-request with same data
		_, restyResp, _ := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.Equal(t, 400, restyResp.StatusCode())
		require.Contains(t, restyResp.String(), "request_already_created")

		// create request by requestedClientID to receivedClientID for different file
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData2)
		require.NoError(t, err, "error creating shareRequestData2")
		reqId2 = dataResp.Data
		toDeleteIds = append(toDeleteIds, reqId2)

		// create request by requestedClientID2 to receivedClientID
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUser2Headers, shareRequestData3)
		require.NoError(t, err, "error creating shareRequestData3")
		reqId3 = dataResp.Data
		toDeleteIds = append(toDeleteIds, reqId3)

		// create request by requestedClientID2 to receivedClientID2
		dataResp, _, err = zboxClient.CreateShareRequest(t, requestedUser2Headers, shareRequestData4)
		require.NoError(t, err, "error creating shareRequestData4")
		reqId4 = dataResp.Data
		toDeleteIds = append(toDeleteIds, reqId4)
	})

	w.RunSequentially("verify get all requests", func(t *test.SystemTest) {
		// verify get all received requests
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, dataResp.Data, 3)
		require.Equal(t, []int64{reqId3, reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID, dataResp.Data[2].ID})

		// verify get all received requests by requestedClientID
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":       "true",
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, dataResp.Data, 2)
		require.Equal(t, []int64{reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all received requests by requestedClientID2
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":       "true",
			"client_id": requestedClientID2,
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reqId3}, []int64{dataResp.Data[0].ID})

		// verify get all received requests by requestedClientID and LookupHash
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"all":         "true",
			"client_id":   requestedClientID,
			"lookup_hash": shareRequestData1["lookup_hash"],
		})
		require.NoError(t, err, "error getting received shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get all share requested by requestedClientID
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, dataResp.Data, 2)
		require.Equal(t, []int64{reqId2, reqId1}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all share requested by requestedClientID with LookupHash filter
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUserHeaders, map[string]string{
			"all":         "true",
			"lookup_hash": shareRequestData1["lookup_hash"],
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get all share requested by requestedClientID2
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUser2Headers, map[string]string{
			"all": "true",
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, dataResp.Data, 2)
		require.Equal(t, []int64{reqId4, reqId3}, []int64{dataResp.Data[0].ID, dataResp.Data[1].ID})

		// verify get all share requested by requestedClientID2 with OwnerID filter
		dataResp, _, err = zboxClient.GetRequestedShareReq(t, requestedUser2Headers, map[string]string{
			"all":      "true",
			"owner_id": receivedClientID2,
		})
		require.NoError(t, err, "error getting requested shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reqId4}, []int64{dataResp.Data[0].ID})
	})

	w.RunSequentially("update requests", func(t *test.SystemTest) {
		// approve reqId1
		dataResp, _, err := zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId1, 10),
			"status": "1",
			"link":   randomLink,
			"auth_ticket": approvalAuthTicket1,
		})
		require.NoError(t, err, "error updating reqId1")
		require.Equal(t, uint8(1), dataResp.Data.Status, "reqId1 should be approved")

		// verify shareinfo gets created
		resp, _, err := zboxClient.GetShareInfoReceivedWithParams(t, requestedUserHeaders, map[string]string{
			"share_info_type": "private",
			"lookup_hash":     shareRequestData1["lookup_hash"],
		})
		require.NoError(t, err, "error verifying shareinfo exists")
		require.Len(t, resp.Data, 1)
		require.Equal(t, resp.Data[0].Receiver, requestedClientID)
		require.Equal(t, approvalAuthTicket1, resp.Data[0].AuthTicket, "share_info authticket should be updated")

		// approve or decline reqId1 again should fail
		_, restyResp, _ := zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId1, 10),
			"status": "2",
		})
		require.Equal(t, 400, restyResp.StatusCode())
		require.Contains(t, restyResp.String(), "status is in final state")

		// approve reqId2 by receivedClientID2 should fail authorization
		_, restyResp, _ = zboxClient.UpdateShareReq(t, receivedUser2Headers, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "1",
		})
		require.Equal(t, 400, restyResp.StatusCode())
		require.Contains(t, restyResp.String(), "unauthorized:")

		// status other than 1 or 2 should fail
		_, restyResp, _ = zboxClient.UpdateShareReq(t, receivedUser2Headers, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "0",
		})
		require.Equal(t, 400, restyResp.StatusCode())
		require.Contains(t, restyResp.String(), "invalid status")

		// decline reqId2
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId2, 10),
			"status": "2",
		})
		require.NoError(t, err, "error updating reqId2")
		require.Equal(t, uint8(2), dataResp.Data.Status, "reqId2 should be declined")

		// decline reqId3
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUserHeaders, map[string]string{
			"id":     strconv.FormatInt(reqId3, 10),
			"status": "2",
		})
		require.NoError(t, err, "error updating reqId3")
		require.Equal(t, uint8(2), dataResp.Data.Status, "reqId3 should be declined")

		// approve reqId4
		dataResp, _, err = zboxClient.UpdateShareReq(t, receivedUser2Headers, map[string]string{
			"id":     strconv.FormatInt(reqId4, 10),
			"status": "1",
			"link":   randomLink,
			"auth_ticket": approvalAuthTicket4,
		})
		require.NoError(t, err, "error updating reqId4")
		require.Equal(t, uint8(1), dataResp.Data.Status, "reqId4 should be approved")
	})

	w.RunSequentially("verify re-request for approved and declined shareReq", func(t *test.SystemTest) {
		// re-request for already approved request should fail
		_, restyResp, _ := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData1)
		require.Equal(t, 400, restyResp.StatusCode())
		require.Contains(t, restyResp.String(), "request_already_approved")

		// re-request for declined request should be allowed
		dataResp, _, err := zboxClient.CreateShareRequest(t, requestedUserHeaders, shareRequestData2)
		require.NoError(t, err, "re-request for declined request should be allowed")
		reReqId = dataResp.Data
		toDeleteIds = append(toDeleteIds, reReqId)
	})

	w.RunSequentially("verify get latest requests", func(t *test.SystemTest) {
		// verify get latest approved request
		dataResp, _, err := zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"status":    "1",
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting latest approved shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reqId1}, []int64{dataResp.Data[0].ID})

		// verify get latest request
		dataResp, _, err = zboxClient.GetReceivedShareReq(t, receivedUserHeaders, map[string]string{
			"client_id": requestedClientID,
		})
		require.NoError(t, err, "error getting latest approved shareReq")
		require.Len(t, dataResp.Data, 1)
		require.Equal(t, []int64{reReqId}, []int64{dataResp.Data[0].ID})
		require.Equal(t, uint8(0), dataResp.Data[0].Status)
	})

}

func convertToAuthTicket(data interface{}) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}
