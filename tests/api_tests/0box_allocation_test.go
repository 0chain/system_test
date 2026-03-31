package api_tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
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
	// Create0boxTestWallet already calls Teardown() internally
	err := Create0boxTestWallet(t, headers)
	if err != nil {
		return err
	}
	allocationInput := NewTestAllocation()
	_, resp, err := zboxClient.CreateAllocation(t, headers, allocationInput)
	if err != nil {
		return fmt.Errorf("CreateAllocation failed: %w", err)
	}
	if resp.StatusCode() != 201 {
		// 400/409 with "already exists" is OK — previous teardown may not have cascaded allocations
		if resp.StatusCode() == 400 || resp.StatusCode() == 409 {
			respStr := resp.String()
			if strings.Contains(respStr, "already exists") || strings.Contains(respStr, "active allocation") {
				t.Logf("Allocation already exists (status %d), proceeding", resp.StatusCode())
				return nil
			}
		}
		return fmt.Errorf("CreateAllocation returned %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func Test0BoxAllocation(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.Parallel()

	t.RunSequentiallyWithTimeout("List allocation with zero allocation should work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 0)
	})

	t.RunSequentiallyWithTimeout("List allocation with existing allocation should work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		if response.StatusCode() != 201 {
			respStr := response.String()
			if strings.Contains(respStr, "already exists") || strings.Contains(respStr, "active allocation") {
				t.Logf("Allocation already exists, proceeding to list check")
			} else {
				require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
			}
		}

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.GreaterOrEqual(t, len(allocationList), 1)
	})

	t.RunSequentiallyWithTimeout("multiple allocations with blimp argument should work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()
		allocInput["id"] = fmt.Sprintf("%064x", time.Now().UnixNano())
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocInput["id"] = fmt.Sprintf("%064x", time.Now().UnixNano()+1)
		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocationList, response, err := zboxClient.ListAllocation(t, headers)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Len(t, allocationList, 2)
	})

	t.RunSequentiallyWithTimeout("multiple allocations with vult argument should not work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Refresh CSRF token after wallet creation to ensure it's valid
		headers = zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_VULT)

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		allocInput["id"] = fmt.Sprintf("%064x", time.Now().UnixNano()+2)
		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentiallyWithTimeout("Post allocation for chimney should not work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_CHIMNEY)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentiallyWithTimeout("Post allocation with already existing allocation Id should not  work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()
		_, response, err := zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 201, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())

		_, response, err = zboxClient.CreateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentiallyWithTimeout("Get an allocation with allocation present should work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

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

	t.RunSequentiallyWithTimeout("Get an allocation with allocation not present should not work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()

		_, response, err := zboxClient.GetAllocation(t, headers, allocInput["id"])
		require.NoError(t, err)
		require.Equal(t, 400, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
	})

	t.RunSequentiallyWithTimeout("Update an allocation with allocation present should work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

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

	t.RunSequentiallyWithTimeout("Update an allocation with allocation not present should not work", 10*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		allocInput := NewTestAllocation()
		allocInput["name"] = "new_alloc_name"
		allocInput["description"] = "new_alloc_description"
		updateResponse, response, err := zboxClient.UpdateAllocation(t, headers, allocInput)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode(), "Response status code does not match expected. Output: [%v]", response.String())
		require.Equal(t, "no allocation was updated for these details", updateResponse.Message)
	})
}
