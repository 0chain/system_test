package api_tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func NewTestAllocation() map[string]string {
	// Generate a unique string using the current timestamp
	uniqueString := fmt.Sprintf("%d", time.Now().UnixNano())

	// Create a new hash
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(uniqueString))

	// Get the hash and convert it to a hexadecimal string
	id := hex.EncodeToString(hasher.Sum(nil))

	return map[string]string{
		"id":              id,
		"description":     "test_allocation_description",
		"name":            "test_allocation_name",
		"allocation_type": "external_drive",
	}
}

func Create0boxTestAllocation(t *test.SystemTest, headers map[string]string) error {
	verifyOtpInput := NewVerifyOtpDetails()
	_, _, err := zboxClient.VerifyOtpDetails(t, headers, verifyOtpInput)
	if err != nil {
		return err
	}
	walletInput := NewTestWallet()
	_, _, err = zboxClient.CreateWallet(t, headers, walletInput)
	if err != nil {
		return err
	}
	allocationInput := NewTestAllocation()
	_, _, err = zboxClient.CreateAllocation(t, headers, allocationInput)
	if err != nil {
		return err
	}
	return nil
}

func Test0BoxAllocation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("List allocation with zero allocation should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 0)
	})

	t.RunSequentially("List allocation with existing allocation should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 1)
	})

	t.RunSequentially("multiple allocations with blimp argument should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		allocInput["id"] = "c0360331837a7376d27007614e124db83811e4416dd2f1577345dd96c8621bf6"
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocInput["id"] = "834d8db33f30952238d9ccc4eb7215ed39752b9686ed858aa7e9653f3d41e79b"
		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 2)
	})

	t.RunSequentially("multiple allocations with vult argument should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocInput["id"] = "834d8db33f30952238d9ccc4eb7215ed39752b9686ed858aa7e9653f3d41e79b"
		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Post allocation for chimney should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_CHIMNEY)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Post allocation with already existing allocation Id should not  work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Get an allocation with allocation present should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocation, response, err := zboxClient.GetAllocation(t, headers, allocInput["id"])
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, allocInput["id"], allocation.ID)
		require.Equal(t, allocInput["description"], allocation.Description)
		require.Equal(t, allocInput["name"], allocation.Name)
		require.Equal(t, allocInput["allocation_type"], allocation.AllocationType)
	})

	t.RunSequentially("Get an allocation with allocation not present should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()

		_, response, err := zboxClient.GetAllocation(t, headers, allocInput["id"])
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentially("Update an allocation with allocation present should work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocInput["name"] = "new_alloc_name"
		allocInput["description"] = "new_alloc_description"
		_, response, err = zboxClient.UpdateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocation, response, err := zboxClient.GetAllocation(t, headers, allocInput["id"])
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, allocInput["id"], allocation.ID)
		require.Equal(t, allocInput["description"], allocation.Description)
		require.Equal(t, allocInput["name"], allocation.Name)
		require.Equal(t, allocInput["allocation_type"], allocation.AllocationType)
	})

	t.RunSequentially("Update an allocation with allocation not present should not work", func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err)

		allocInput := NewTestAllocation()
		allocInput["name"] = "new_alloc_name"
		allocInput["description"] = "new_alloc_description"
		updateResponse, response, err := zboxClient.UpdateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no allocation was updated for these details", updateResponse.Message)
	})
}
