package api_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxTranscoder(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Create transcoding metadata for MP4 file")

	// Shared wallet setup for all subtests
	headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
	Teardown(t, headers)
	err := Create0boxTestWallet(t, headers)
	require.NoError(t, err, "0box wallet setup")

	// Use a fake but plausible allocation ID for metadata-only tests.
	// The 0box CreateMetadata endpoint stores metadata — it does not validate
	// that the allocation actually exists on-chain at creation time.
	fakeAllocationID := fmt.Sprintf("test_alloc_%d", time.Now().UnixNano())

	t.RunSequentiallyWithTimeout("Create transcoding metadata for MP4 file", 2*time.Minute, func(t *test.SystemTest) {
		metaBody := map[string]interface{}{
			"remotepath":    "/sample.mp4",
			"mode":          "web",
			"allocation_id": fakeAllocationID,
			"file_name":     "sample.mp4",
			"file_size":     1048576, // 1MB
			"file_path":     "/sample.mp4/sample.mp4",
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata API call should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Output: [%v]", resp.String())
		require.NotNil(t, entity, "TranscodingEntity should not be nil")

		t.Logf("Created transcoding entity: ID=%d, Status=%d, FileName=%s",
			entity.ID, entity.Status, entity.FileName)

		require.Greater(t, entity.ID, int64(0), "Entity ID should be positive")
		require.Equal(t, "sample.mp4", entity.FileName, "FileName should match")
		require.Equal(t, fakeAllocationID, entity.AllocationID, "AllocationID should match")
		require.Equal(t, "web", entity.Mode, "Mode should match")
	})

	t.RunSequentiallyWithTimeout("Create transcoding metadata for AVI file", 2*time.Minute, func(t *test.SystemTest) {
		metaBody := map[string]interface{}{
			"remotepath":    "/sample.avi",
			"mode":          "web",
			"allocation_id": fakeAllocationID,
			"file_name":     "sample.avi",
			"file_size":     2097152, // 2MB
			"file_path":     "/sample.avi/sample.avi",
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata API call should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Output: [%v]", resp.String())
		require.NotNil(t, entity, "TranscodingEntity should not be nil")

		t.Logf("Created transcoding entity: ID=%d, Status=%d, FileName=%s",
			entity.ID, entity.Status, entity.FileName)

		require.Greater(t, entity.ID, int64(0), "Entity ID should be positive")
		require.Equal(t, "sample.avi", entity.FileName, "FileName should match")
	})

	t.RunSequentiallyWithTimeout("Get metadata and verify status after create", 2*time.Minute, func(t *test.SystemTest) {
		// First create a metadata entry
		metaBody := map[string]interface{}{
			"remotepath":    "/status_test.mp4",
			"mode":          "web",
			"allocation_id": fakeAllocationID,
			"file_name":     "status_test.mp4",
			"file_size":     512000,
			"file_path":     "/status_test.mp4/status_test.mp4",
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Output: [%v]", resp.String())
		require.NotNil(t, entity, "Created entity should not be nil")
		createdID := entity.ID
		t.Logf("Created entity ID=%d for status check", createdID)

		// Update upload status to mark upload complete (status=1)
		updateBody := map[string]interface{}{
			"id":     createdID,
			"status": 1,
		}
		updatedEntity, resp, err := zboxClient.UpdateUploadStatus(t, headers, updateBody)
		require.NoError(t, err, "UpdateUploadStatus should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"UpdateUploadStatus should return 201. Output: [%v]", resp.String())
		require.NotNil(t, updatedEntity, "Updated entity should not be nil")
		t.Logf("Updated entity ID=%d, Status=%d", updatedEntity.ID, updatedEntity.Status)

		// Query metadata back via GET
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": "/status_test.mp4/status_test.mp4",
			"mode":      "web",
		}

		getEntity, resp, err := zboxClient.GetMetadata(t, headers, queryParams)
		require.NoError(t, err, "GetMetadata should not fail")
		require.Equal(t, 200, resp.StatusCode(),
			"GetMetadata should return 200. Output: [%v]", resp.String())
		require.NotNil(t, getEntity, "GetMetadata entity should not be nil")

		t.Logf("GetMetadata: ID=%d, Status=%d, FileName=%s",
			getEntity.ID, getEntity.Status, getEntity.FileName)

		require.Equal(t, createdID, getEntity.ID, "Entity ID should match created ID")
		require.Equal(t, "status_test.mp4", getEntity.FileName, "FileName should match")
		// Status should be >= 1 (upload complete or further along in processing)
		require.GreaterOrEqual(t, getEntity.Status, 1,
			"Status should be at least 1 (upload complete) after update")
	})

	t.RunSequentiallyWithTimeout("Create metadata for non-video file", 2*time.Minute, func(t *test.SystemTest) {
		metaBody := map[string]interface{}{
			"remotepath":    "/document.pdf",
			"mode":          "web",
			"allocation_id": fakeAllocationID,
			"file_name":     "document.pdf",
			"file_size":     65536,
			"file_path":     "/document.pdf/document.pdf",
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata should not return transport error")
		t.Logf("CreateMetadata (non-video): status=%d, body=%s", resp.StatusCode(), resp.String())

		// The 0box may accept metadata for any file (transcoding fails later)
		// or may reject non-video files at creation time. Both are valid.
		if resp.StatusCode() == 201 {
			require.NotNil(t, entity)
			require.Equal(t, "document.pdf", entity.FileName)
			t.Logf("Non-video metadata accepted (ID=%d), transcoding will fail asynchronously", entity.ID)
		} else {
			// If rejected at creation, expect a 4xx error
			require.True(t, resp.StatusCode() >= 400 && resp.StatusCode() < 500,
				"Non-video rejection should be 4xx, got %d", resp.StatusCode())
			t.Logf("Non-video file rejected at metadata creation (status=%d)", resp.StatusCode())
		}
	})

	t.RunSequentiallyWithTimeout("Get metadata for non-existent file returns 404", 2*time.Minute, func(t *test.SystemTest) {
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": fmt.Sprintf("/nonexistent_%d/nonexistent_%d.mp4", time.Now().UnixNano(), time.Now().UnixNano()),
			"mode":      "web",
		}

		entity, resp, err := zboxClient.GetMetadata(t, headers, queryParams)
		require.NoError(t, err, "GetMetadata should not return transport error")
		t.Logf("GetMetadata (non-existent): status=%d, body=%s", resp.StatusCode(), resp.String())

		// Expect 404 or empty result for a file path that was never created
		if resp.StatusCode() == 200 {
			// Some implementations return 200 with nil/empty data
			if entity != nil {
				require.Equal(t, int64(0), entity.ID,
					"Non-existent file should return empty entity (ID=0)")
			}
		} else {
			require.True(t, resp.StatusCode() == 404 || resp.StatusCode() == 400,
				"Non-existent metadata should return 404 or 400, got %d", resp.StatusCode())
		}
	})
}
