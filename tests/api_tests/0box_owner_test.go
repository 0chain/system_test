package api_tests

import (
	"testing"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Teardown(t *test.SystemTest, headers map[string]string) {
	t.Logf("Tearing down existing data")
	message, response, err := zboxClient.DeleteOwner(t, headers)
	println(message, response, err)
}

func NewTestOwner() map[string]string {
	return map[string]string{
		"username":     "test_owner_1",
		"email":        "test_email_1",
		"phone_number": "+919876543210",
	}
}

func NewVerifyOtpDetails() map[string]string {
	return map[string]string{
		"username":       "test_owner_1",
		"email":          "test_email_1",
		"phone_number":   "+919876543210",
		"otp":            "123456",
		"firebase_token": "test_firebase_token",
		"user_id":        client.X_APP_USER_ID,
	}
}

func Test0BoxOwner(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("create owner without existing userID should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, response, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		owner, response, err := zboxClient.GetOwner(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, verifyOtpInput["username"], owner.UserName)
		require.Equal(t, verifyOtpInput["email"], owner.Email)
		require.Equal(t, verifyOtpInput["phone_number"], owner.PhoneNumber)
	})

	t.RunSequentially("create owner with existing userID should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		_, response, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("update owner with existing owner should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		verifyOtpInput := NewVerifyOtpDetails()
		_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
		require.NoError(t, err)

		ownerInput := NewTestOwner()
		ownerInput["username"] = "new_user_name"
		ownerInput["biography"] = "new_biography"
		message, _, err := zboxClient.UpdateOwner(t, headers, ownerInput)
		require.NoError(t, err)
		require.Equal(t, "updated owner details successfully", message.Message)

		owner, response, err := zboxClient.GetOwner(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, ownerInput["username"], owner.UserName)
		require.Equal(t, ownerInput["email"], owner.Email)
		require.Equal(t, ownerInput["phone_number"], owner.PhoneNumber)
		require.Equal(t, ownerInput["biography"], owner.Biography)
	})

	t.RunSequentially("update owner without existing owner should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeaders(client.X_APP_BLIMP)
		Teardown(t, headers)

		ownerInput := NewTestOwner()
		message, _, err := zboxClient.UpdateOwner(t, headers, ownerInput)
		require.NoError(t, err)
		require.Equal(t, "No Data was updated", message.Message)
	})
}
