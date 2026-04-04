package api_tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// Test0BoxGroup tests the Groups API endpoints:
// - POST   /v2/groups           (create group)
// - GET    /v2/groups/my        (get my groups)
// - POST   /v2/groups/:id/members (add member)
// - GET    /v2/groups/:id/members (get members)
// - PUT    /v2/groups/:id       (update group)
// - DELETE /v2/groups/:id       (delete group)
// - GET    /v2/groups/search    (search public groups)
func Test0BoxGroup(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)

	t.RunSequentiallyWithTimeout("Create group should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		body := map[string]interface{}{
			"group_name":  "test-group-create",
			"description": "Integration test group",
			"is_public":   true,
			"max_members": 50,
		}

		resp, err := zboxClient.CreateGroup(t, headers, body)
		require.NoError(t, err)
		t.Logf("CreateGroup: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 201, resp.StatusCode(),
			"Group creation should return 201. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Create group without group_name should fail validation", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		body := map[string]interface{}{
			"description": "Missing group_name",
			"is_public":   true,
		}

		resp, err := zboxClient.CreateGroup(t, headers, body)
		require.NoError(t, err)
		t.Logf("CreateGroup (no name): %d - %s", resp.StatusCode(), resp.String())
		require.True(t, resp.StatusCode() == 400 || resp.StatusCode() == 500,
			"Missing group_name should return 400 or 500. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Get my groups should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a group first
		body := map[string]interface{}{
			"group_name":  "test-group-my",
			"description": "Group for GetMyGroups test",
			"is_public":   false,
			"max_members": 10,
		}
		resp, err := zboxClient.CreateGroup(t, headers, body)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode(),
			"Group creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Get my groups
		resp, err = zboxClient.GetMyGroups(t, headers)
		require.NoError(t, err)
		t.Logf("GetMyGroups: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 200, resp.StatusCode(),
			"GetMyGroups should return 200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Add member to group should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a group
		createBody := map[string]interface{}{
			"group_name":  "test-group-add-member",
			"description": "Group for AddMember test",
			"is_public":   true,
			"max_members": 50,
		}
		resp, err := zboxClient.CreateGroup(t, headers, createBody)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode(),
			"Group creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Extract group ID from response
		var groupResp map[string]interface{}
		err = json.Unmarshal(resp.Body(), &groupResp)
		require.NoError(t, err, "Failed to parse create group response")
		groupID := fmt.Sprintf("%.0f", groupResp["id"].(float64))
		t.Logf("Created group ID: %s", groupID)

		// Add a member (wallet_address max 64 chars)
		memberBody := map[string]interface{}{
			"user_id":        "test_member_user_123",
			"wallet_address": "abcdef1234567890abcdef1234567890",
			"member_name":    "Test Member",
			"role":           "member",
		}
		resp, err = zboxClient.AddGroupMember(t, headers, groupID, memberBody)
		require.NoError(t, err)
		t.Logf("AddGroupMember: %d - %s", resp.StatusCode(), resp.String())
		require.True(t, resp.StatusCode() == 201 || resp.StatusCode() == 200,
			"AddGroupMember should return 201/200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Get group members should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a group
		createBody := map[string]interface{}{
			"group_name":  "test-group-get-members",
			"description": "Group for GetGroupMembers test",
			"is_public":   true,
			"max_members": 50,
		}
		resp, err := zboxClient.CreateGroup(t, headers, createBody)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode())

		var groupResp map[string]interface{}
		err = json.Unmarshal(resp.Body(), &groupResp)
		require.NoError(t, err)
		groupID := fmt.Sprintf("%.0f", groupResp["id"].(float64))

		// Get members (owner is auto-added as first member)
		resp, err = zboxClient.GetGroupMembers(t, headers, groupID)
		require.NoError(t, err)
		t.Logf("GetGroupMembers: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 200, resp.StatusCode(),
			"GetGroupMembers should return 200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Update group should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a group
		createBody := map[string]interface{}{
			"group_name":  "test-group-update",
			"description": "Original description",
			"is_public":   false,
			"max_members": 10,
		}
		resp, err := zboxClient.CreateGroup(t, headers, createBody)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode())

		var groupResp map[string]interface{}
		err = json.Unmarshal(resp.Body(), &groupResp)
		require.NoError(t, err)
		groupID := fmt.Sprintf("%.0f", groupResp["id"].(float64))

		// Update the group
		updateBody := map[string]interface{}{
			"group_name":  "test-group-updated",
			"description": "Updated description",
		}
		resp, err = zboxClient.UpdateGroup(t, headers, groupID, updateBody)
		require.NoError(t, err)
		t.Logf("UpdateGroup: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 200, resp.StatusCode(),
			"UpdateGroup should return 200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Delete group should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a group
		createBody := map[string]interface{}{
			"group_name":  "test-group-delete",
			"description": "Group to be deleted",
			"is_public":   true,
			"max_members": 5,
		}
		resp, err := zboxClient.CreateGroup(t, headers, createBody)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode())

		var groupResp map[string]interface{}
		err = json.Unmarshal(resp.Body(), &groupResp)
		require.NoError(t, err)
		groupID := fmt.Sprintf("%.0f", groupResp["id"].(float64))

		// Delete the group
		resp, err = zboxClient.DeleteGroup(t, headers, groupID)
		require.NoError(t, err)
		t.Logf("DeleteGroup: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 200, resp.StatusCode(),
			"DeleteGroup should return 200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Search public groups should work", 3*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Create a public group to search for
		createBody := map[string]interface{}{
			"group_name":  "searchable-test-group",
			"description": "A public group for search test",
			"is_public":   true,
			"max_members": 50,
		}
		resp, err := zboxClient.CreateGroup(t, headers, createBody)
		require.NoError(t, err)
		require.Equal(t, 201, resp.StatusCode(),
			"Group creation should succeed. Got %d: %s", resp.StatusCode(), resp.String())

		// Search for the group
		resp, err = zboxClient.SearchGroups(t, headers, "searchable")
		require.NoError(t, err)
		t.Logf("SearchGroups: %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 200, resp.StatusCode(),
			"SearchGroups should return 200. Got %d: %s", resp.StatusCode(), resp.String())
	})

	t.RunSequentiallyWithTimeout("Search groups without query should fail", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)

		// Search with empty query
		resp, err := zboxClient.SearchGroups(t, headers, "")
		require.NoError(t, err)
		t.Logf("SearchGroups (empty): %d - %s", resp.StatusCode(), resp.String())
		require.Equal(t, 400, resp.StatusCode(),
			"Empty search query should return 400. Got %d", resp.StatusCode())
	})

	t.RunSequentiallyWithTimeout("Delete non-existent group should return appropriate response", 2*time.Minute, func(t *test.SystemTest) {
		headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
		Teardown(t, headers)

		err := Create0boxTestWallet(t, headers)
		require.NoError(t, err, "0box wallet setup")

		// Try to delete a group that does not exist
		resp, err := zboxClient.DeleteGroup(t, headers, "999999")
		require.NoError(t, err)
		t.Logf("DeleteGroup (non-existent): %d - %s", resp.StatusCode(), resp.String())
		// 200 with "no group was deleted" message is the expected response from the handler
		require.True(t, resp.StatusCode() == 200 || resp.StatusCode() == 400 || resp.StatusCode() == 404,
			"Delete non-existent group should return 200/400/404. Got %d", resp.StatusCode())
	})
}
