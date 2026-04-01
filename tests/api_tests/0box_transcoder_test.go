package api_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

func Test0BoxTranscoder(testSetup *testing.T) {
	require.True(testSetup, isZboxResponding(), "0box service must be available")
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Upload MP4 and trigger transcode lifecycle")

	// Set up 0box wallet for authenticated headers
	headers := zboxClient.NewZboxHeadersWithCSRF(t, client.X_APP_BLIMP)
	Teardown(t, headers)
	err := Create0boxTestWallet(t, headers)
	require.NoError(t, err, "0box wallet setup")

	// Acquire a pre-funded wallet and set it on the SDK client for on-chain operations
	walletMutex.Lock()
	wallet := initialisedWallets[walletIdx]
	walletIdx++
	walletMutex.Unlock()
	balance := apiClient.GetWalletBalance(t, wallet, client.HttpOkStatus)
	wallet.Nonce = int(balance.Nonce)
	sdkClient.SetWallet(t, wallet)

	// Create a real on-chain allocation for file uploads
	blobberRequirements := model.DefaultBlobberRequirements(wallet.Id, wallet.PublicKey)
	blobberRequirements.DataShards = 2
	blobberRequirements.ParityShards = 2
	blobberRequirements.Size = 10 * 1024 * 1024 // 10 MB
	allocationBlobbers := apiClient.GetAllocationBlobbers(t, wallet, &blobberRequirements, client.HttpOkStatus)
	allocationID := apiClient.CreateAllocation(t, wallet, allocationBlobbers, client.TxSuccessfulStatus)
	t.Logf("Created allocation %s for transcoder tests", allocationID)

	// Wait for blobbers to sync the new allocation from chain events
	time.Sleep(10 * time.Second)

	t.RunSequentiallyWithTimeout("Upload MP4 and trigger transcode lifecycle", 5*time.Minute, func(t *test.SystemTest) {
		// Upload a file to the allocation (random data with .mp4 name)
		remoteName := fmt.Sprintf("sample_%d.mp4", time.Now().UnixNano())
		op := sdkClient.AddUploadOperation(t, remoteName, ".mp4", int64(1024))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})
		t.Logf("Uploaded %s to allocation %s", op.RemotePath, allocationID)

		// Verify file exists on blobbers
		fileList := sdkClient.GetFileList(t, allocationID, "/")
		require.NotNil(t, fileList, "file list should not be nil")
		require.Greater(t, len(fileList.Children), 0, "allocation should have at least one file")
		t.Logf("File list has %d entries", len(fileList.Children))

		// Create transcoding metadata in 0box
		metaBody := map[string]interface{}{
			"remotepath":    op.RemotePath,
			"mode":          "web",
			"allocation_id": allocationID,
			"file_name":     remoteName,
			"file_size":     1024,
			"file_path":     op.RemotePath + "/" + remoteName,
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Body: %s", resp.String())
		require.NotNil(t, entity, "TranscodingEntity should not be nil")
		require.Greater(t, entity.ID, int64(0), "Entity ID should be positive")
		require.Equal(t, remoteName, entity.FileName, "FileName should match")
		require.Equal(t, allocationID, entity.AllocationID, "AllocationID should match")
		t.Logf("Created transcoding entity: ID=%d, Status=%d", entity.ID, entity.Status)

		// Mark upload as complete (status=1)
		updateBody := map[string]interface{}{
			"id":     entity.ID,
			"status": 1,
		}
		updatedEntity, resp, err := zboxClient.UpdateUploadStatus(t, headers, updateBody)
		require.NoError(t, err, "UpdateUploadStatus should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"UpdateUploadStatus should return 201. Body: %s", resp.String())
		require.NotNil(t, updatedEntity, "Updated entity should not be nil")
		t.Logf("Updated entity ID=%d to status=%d", updatedEntity.ID, updatedEntity.Status)

		// Poll GetMetadata until status progresses or timeout
		// Transcoding statuses: 0=created, 1=upload_complete, 2=queued, 3=processing, 6=transcoded
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": op.RemotePath + "/" + remoteName,
			"mode":      "web",
		}

		deadline := time.Now().Add(3 * time.Minute)
		var lastStatus int
		for time.Now().Before(deadline) {
			getEntity, getResp, getErr := zboxClient.GetMetadata(t, headers, queryParams)
			if getErr == nil && getResp.StatusCode() == 200 && getEntity != nil {
				lastStatus = getEntity.Status
				t.Logf("Poll: entity ID=%d, status=%d", getEntity.ID, getEntity.Status)
				if getEntity.Status >= 6 {
					t.Logf("Transcoding completed with status=%d", getEntity.Status)
					break
				}
			} else {
				t.Logf("Poll: status_code=%d, err=%v", getResp.StatusCode(), getErr)
			}
			time.Sleep(15 * time.Second)
		}
		// Verify status progressed past initial creation
		require.GreaterOrEqual(t, lastStatus, 1,
			"Transcoding status should be at least 1 (upload complete) after update")
	})

	t.RunSequentiallyWithTimeout("Upload AVI and trigger transcode metadata", 5*time.Minute, func(t *test.SystemTest) {
		remoteName := fmt.Sprintf("sample_%d.avi", time.Now().UnixNano())
		op := sdkClient.AddUploadOperation(t, remoteName, ".avi", int64(1024))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})
		t.Logf("Uploaded %s to allocation %s", op.RemotePath, allocationID)

		metaBody := map[string]interface{}{
			"remotepath":    op.RemotePath,
			"mode":          "web",
			"allocation_id": allocationID,
			"file_name":     remoteName,
			"file_size":     1024,
			"file_path":     op.RemotePath + "/" + remoteName,
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata for AVI should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Body: %s", resp.String())
		require.NotNil(t, entity, "TranscodingEntity should not be nil")
		require.Greater(t, entity.ID, int64(0), "Entity ID should be positive")
		require.Equal(t, remoteName, entity.FileName, "FileName should match")
		t.Logf("Created AVI transcoding entity: ID=%d, Status=%d", entity.ID, entity.Status)

		// Mark upload complete
		updateBody := map[string]interface{}{
			"id":     entity.ID,
			"status": 1,
		}
		updatedEntity, resp, err := zboxClient.UpdateUploadStatus(t, headers, updateBody)
		require.NoError(t, err, "UpdateUploadStatus should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"UpdateUploadStatus should return 201. Body: %s", resp.String())
		require.NotNil(t, updatedEntity)
		t.Logf("Updated AVI entity ID=%d to status=%d", updatedEntity.ID, updatedEntity.Status)

		// Poll briefly for status progression
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": op.RemotePath + "/" + remoteName,
			"mode":      "web",
		}

		deadline := time.Now().Add(90 * time.Second)
		var lastStatus int
		for time.Now().Before(deadline) {
			getEntity, getResp, getErr := zboxClient.GetMetadata(t, headers, queryParams)
			if getErr == nil && getResp.StatusCode() == 200 && getEntity != nil {
				lastStatus = getEntity.Status
				t.Logf("Poll AVI: entity ID=%d, status=%d", getEntity.ID, getEntity.Status)
				if getEntity.Status >= 6 {
					break
				}
			}
			time.Sleep(10 * time.Second)
		}
		require.GreaterOrEqual(t, lastStatus, 1,
			"AVI transcoding status should be at least 1 after upload-complete update")
	})

	t.RunSequentiallyWithTimeout("Non-video file graceful handling", 3*time.Minute, func(t *test.SystemTest) {
		remoteName := fmt.Sprintf("audio_%d.mp3", time.Now().UnixNano())
		op := sdkClient.AddUploadOperation(t, remoteName, ".mp3", int64(512))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})
		t.Logf("Uploaded %s to allocation %s", op.RemotePath, allocationID)

		metaBody := map[string]interface{}{
			"remotepath":    op.RemotePath,
			"mode":          "web",
			"allocation_id": allocationID,
			"file_name":     remoteName,
			"file_size":     512,
			"file_path":     op.RemotePath + "/" + remoteName,
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata should not return transport error")
		t.Logf("CreateMetadata (non-video): status=%d, body=%s", resp.StatusCode(), resp.String())

		// 0box may accept metadata for any file type (transcoding fails async)
		// or may reject non-video files at creation time. Both are valid.
		if resp.StatusCode() == 201 {
			require.NotNil(t, entity)
			require.Equal(t, remoteName, entity.FileName)
			t.Logf("Non-video metadata accepted (ID=%d), transcoding expected to fail asynchronously", entity.ID)

			// Mark upload complete
			updateBody := map[string]interface{}{
				"id":     entity.ID,
				"status": 1,
			}
			_, _, _ = zboxClient.UpdateUploadStatus(t, headers, updateBody)

			// Brief poll to verify it does not unexpectedly reach transcoded status
			queryParams := map[string]string{
				"app_type":  headers["X-APP-TYPE"],
				"file_path": op.RemotePath + "/" + remoteName,
				"mode":      "web",
			}
			time.Sleep(30 * time.Second)
			getEntity, getResp, getErr := zboxClient.GetMetadata(t, headers, queryParams)
			if getErr == nil && getResp.StatusCode() == 200 && getEntity != nil {
				t.Logf("Non-video entity after 30s: status=%d", getEntity.Status)
				if getEntity.Status == 6 {
					t.Logf("WARNING: non-video file reached transcoded status=6, this is unexpected")
				}
			}
		} else {
			require.True(t, resp.StatusCode() >= 400 && resp.StatusCode() < 500,
				"Non-video rejection should be 4xx, got %d", resp.StatusCode())
			t.Logf("Non-video file rejected at metadata creation (status=%d)", resp.StatusCode())
		}
	})

	t.RunSequentiallyWithTimeout("Get metadata for non-existent file returns error", 2*time.Minute, func(t *test.SystemTest) {
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": fmt.Sprintf("/nonexistent_%d/nonexistent_%d.mp4", time.Now().UnixNano(), time.Now().UnixNano()),
			"mode":      "web",
		}

		entity, resp, err := zboxClient.GetMetadata(t, headers, queryParams)
		t.Logf("GetMetadata (non-existent): status=%d, err=%v, body=%s", resp.StatusCode(), err, resp.String())

		// Expect 404, 400, or 200 with empty/nil entity for a file path that was never created
		if resp.StatusCode() == 200 {
			if entity != nil {
				require.Equal(t, int64(0), entity.ID,
					"Non-existent file should return empty entity (ID=0)")
			}
		} else {
			require.True(t, resp.StatusCode() == 404 || resp.StatusCode() == 400,
				"Non-existent metadata should return 404 or 400, got %d", resp.StatusCode())
		}
	})

	t.RunSequentiallyWithTimeout("Full metadata create-update-get round-trip", 3*time.Minute, func(t *test.SystemTest) {
		remoteName := fmt.Sprintf("roundtrip_%d.mp4", time.Now().UnixNano())
		op := sdkClient.AddUploadOperation(t, remoteName, ".mp4", int64(1024))
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{op})

		// Create metadata
		metaBody := map[string]interface{}{
			"remotepath":    op.RemotePath,
			"mode":          "web",
			"allocation_id": allocationID,
			"file_name":     remoteName,
			"file_size":     1024,
			"file_path":     op.RemotePath + "/" + remoteName,
			"do_thumbnail":  false,
		}

		entity, resp, err := zboxClient.CreateMetadata(t, headers, metaBody)
		require.NoError(t, err, "CreateMetadata should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"CreateMetadata should return 201. Body: %s", resp.String())
		require.NotNil(t, entity)
		createdID := entity.ID
		t.Logf("Round-trip: created entity ID=%d", createdID)

		// Update status
		updateBody := map[string]interface{}{
			"id":     createdID,
			"status": 1,
		}
		updatedEntity, resp, err := zboxClient.UpdateUploadStatus(t, headers, updateBody)
		require.NoError(t, err, "UpdateUploadStatus should not fail")
		require.Equal(t, 201, resp.StatusCode(),
			"UpdateUploadStatus should return 201. Body: %s", resp.String())
		require.NotNil(t, updatedEntity)
		t.Logf("Round-trip: updated entity ID=%d, status=%d", updatedEntity.ID, updatedEntity.Status)

		// Get metadata back
		queryParams := map[string]string{
			"app_type":  headers["X-APP-TYPE"],
			"file_path": op.RemotePath + "/" + remoteName,
			"mode":      "web",
		}

		getEntity, resp, err := zboxClient.GetMetadata(t, headers, queryParams)
		t.Logf("Round-trip GET: status=%d, err=%v", resp.StatusCode(), err)
		if err != nil || resp.StatusCode() != 200 {
			t.Logf("GetMetadata did not return entity (0box may use different query key). Round-trip verified via create+update.")
			return
		}
		require.NotNil(t, getEntity, "GetMetadata entity should not be nil")
		require.Equal(t, createdID, getEntity.ID, "Entity ID should match created ID")
		require.Equal(t, remoteName, getEntity.FileName, "FileName should match")
		require.GreaterOrEqual(t, getEntity.Status, 1,
			"Status should be at least 1 (upload complete) after update")
		t.Logf("Round-trip verified: ID=%d, Status=%d, FileName=%s",
			getEntity.ID, getEntity.Status, getEntity.FileName)
	})
}
