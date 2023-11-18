package api_tests

import (
	"crypto/rand"
	"math/big"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

var fileExtensions = []string{
	".txt", ".docx", ".pdf", ".jpg", ".png", ".mp3", ".mp4", ".xlsx", ".html", ".json",
	".csv", ".xml", ".zip", ".rar", ".gz", ".tar", ".avi", ".mov", ".wav", ".ogg", ".bmp",
	".gif", ".svg", ".tiff", ".ico", ".py", ".c", ".java", ".php", ".js", ".css", ".scss",
	".yaml", ".sql", ".md", ".go", ".rb", ".cpp", ".h", ".sh", ".bat", ".dll", ".class",
	".jar", ".exe", ".psd", ".pptx", ".xls", ".ppt", ".key", ".numbers", ".m4a", ".flv",
	".html", ".jsp", ".jspf", ".jspx", ".wma", ".wmv", ".asf", ".mov", ".qt", ".fla",
	".swf", ".ai", ".indd", ".dwg", ".dxf", ".eps", ".rtf", ".aac", ".ac3", ".m4v", ".vob",
	".3gp", ".webm", ".tif", ".jfif", ".jp2", ".jpx", ".jb2", ".j2k", ".jpf", ".dds", ".raw",
	".webp", ".svgz", ".eps", ".ai", ".odt", ".ott", ".sxw", ".stw", ".odp", ".otp", ".sxi",
	".sti", ".odg", ".otg", ".sxd",
}

func TestMultiOperation(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Multi upload operations should work")

	t.RunSequentially("Multi upload operations should work", func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 1*KB, 10, 10, false, []int64{}, "")
	})

	t.RunSequentiallyWithTimeout("Multi upload operations of single format should work with 50 large and 50 small files", 500*time.Minute, func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 2*GB, 100, 100, false, []int64{1 * KB, 40 * MB}, "")
	})

	t.RunSequentiallyWithTimeout("Multi upload operations of multiple formats should work with 50 large and 50 small files", 500*time.Minute, func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 2*GB, 100, 100, true, []int64{1 * KB, 40 * MB}, "")
	})

	t.RunSequentially("Multi delete operations should work", func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 1*KB, 10, 0, false, []int64{}, "delete")
	})

	t.RunSequentially("Multi update operations should work", func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 1*KB, 10, 10, false, []int64{}, "update")
	})

	t.RunSequentially("Multi rename operations should work", func(t *test.SystemTest) {
		createAllocationAndPerformMultiOperation(t, 1*KB, 10, 10, false, []int64{}, "rename")
	})

	t.RunSequentially("Multi different operations should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			switch i % 3 {
			case 0:
				op := sdkClient.AddDeleteOperation(t, allocationID, ops[i].FileMeta.RemotePath)
				newOps = append(newOps, op)
			case 1:
				op := sdkClient.AddUpdateOperation(t, allocationID, ops[i].FileMeta.RemotePath, ops[i].FileMeta.RemoteName)
				newOps = append(newOps, op)
			case 2:
				op := sdkClient.AddRenameOperation(t, allocationID, ops[i].FileMeta.RemotePath, randName())
				newOps = append(newOps, op)
			}
		}

		start := time.Now()
		sdkClient.MultiOperation(t, allocationID, newOps)
		end := time.Since(start)
		t.Logf("Multi different operations took %v", end)

		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 6, len(listResult.Children), "files count mismatch expected %v actual %v", 6, len(listResult.Children))
	})

	t.RunSequentially("Multi move operations should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				newPath := "/new/" + filepath.Join("", filepath.Base(ops[i].FileMeta.Path))
				op := sdkClient.AddMoveOperation(t, allocationID, ops[i].FileMeta.RemotePath, newPath)
				newOps = append(newOps, op)
			} else {
				newPath := "/child/" + filepath.Join("", filepath.Base(ops[i].FileMeta.Path))
				op := sdkClient.AddMoveOperation(t, allocationID, ops[i].FileMeta.RemotePath, newPath)
				newOps = append(newOps, op)
			}
		}

		start := time.Now()
		sdkClient.MultiOperation(t, allocationID, newOps)
		end := time.Since(start)
		t.Logf("Multi move operations took %v", end)

		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 2, len(listResult.Children), "files count mismatch expected %v actual %v", 2, len(listResult.Children))

		listResult = sdkClient.GetFileList(t, allocationID, "/new")
		require.Equal(t, 5, len(listResult.Children), "files count mismatch expected %v actual %v", 5, len(listResult.Children))
		listResult = sdkClient.GetFileList(t, allocationID, "/child")
		require.Equal(t, 5, len(listResult.Children), "files count mismatch expected %v actual %v", 5, len(listResult.Children))
	})

	t.RunSequentially("Multi copy operations should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddUploadOperation(t, "")
			ops = append(ops, op)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				newPath := "/new/" + filepath.Join("", filepath.Base(ops[i].FileMeta.Path))
				op := sdkClient.AddCopyOperation(t, allocationID, ops[i].FileMeta.RemotePath, newPath)
				newOps = append(newOps, op)
			} else {
				newPath := "/child/" + filepath.Join("", filepath.Base(ops[i].FileMeta.Path))
				op := sdkClient.AddCopyOperation(t, allocationID, ops[i].FileMeta.RemotePath, newPath)
				newOps = append(newOps, op)
			}
		}

		start := time.Now()
		sdkClient.MultiOperation(t, allocationID, newOps)
		end := time.Since(start)
		t.Logf("Multi copy operations took %v", end)

		listResult := sdkClient.GetFileList(t, allocationID, "/")
		require.Equal(t, 12, len(listResult.Children), "files count mismatch expected %v actual %v", 12, len(listResult.Children))
		listResult = sdkClient.GetFileList(t, allocationID, "/new")
		require.Equal(t, 5, len(listResult.Children), "files count mismatch expected %v actual %v", 5, len(listResult.Children))
		listResult = sdkClient.GetFileList(t, allocationID, "/child")
		require.Equal(t, 5, len(listResult.Children), "files count mismatch expected %v actual %v", 5, len(listResult.Children))
	})

	t.RunSequentially("Multi create dir operations should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		ops := make([]sdk.OperationRequest, 0, 10)
		names := make([]string, 0, 10)
		for i := 0; i < 10; i++ {
			name := path.Join("/", randName())
			op := sdkClient.AddCreateDirOperation(t, allocationID, name)
			ops = append(ops, op)
			names = append(names, name)
		}
		sdkClient.MultiOperation(t, allocationID, ops)

		newOps := make([]sdk.OperationRequest, 0, 10)

		for i := 0; i < 10; i++ {
			op := sdkClient.AddCreateDirOperation(t, allocationID, path.Join(names[i], randName()))
			newOps = append(newOps, op)
		}

		sdkClient.MultiOperation(t, allocationID, newOps)
	})

	t.RunSequentially("Nested move operation should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		nestedDir := sdkClient.AddUploadOperationWithPath(t, allocationID, "/new/nested/")

		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{nestedDir})

		newPath := "/child"
		moveOp := sdkClient.AddMoveOperation(t, allocationID, filepath.Dir(nestedDir.FileMeta.RemotePath), newPath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{moveOp})
		listResult := sdkClient.GetFileList(t, allocationID, "/child/")
		require.Equal(t, 1, len(listResult.Children), "files count mismatch expected %v actual %v", 1, len(listResult.Children))
		listResult = sdkClient.GetFileList(t, allocationID, "/new/")
		require.Equal(t, 0, len(listResult.Children), "files count mismatch expected %v actual %v", 0, len(listResult.Children))
	})

	t.RunSequentially("Nested copy operation should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)

		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		nestedDir := sdkClient.AddUploadOperationWithPath(t, allocationID, "/new/nested/")

		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{nestedDir})

		newPath := "/child"
		copyOp := sdkClient.AddCopyOperation(t, allocationID, filepath.Dir(nestedDir.FileMeta.RemotePath), newPath)
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{copyOp})

		listResult := sdkClient.GetFileList(t, allocationID, "/child/")
		require.Equal(t, 1, len(listResult.Children), "files count mismatch expected %v actual %v", 1, len(listResult.Children))
	})

	t.RunSequentially("Nested rename directory operation should work", func(t *test.SystemTest) {
		apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
		blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
		allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
		allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

		nestedDir := sdkClient.AddCreateDirOperation(t, allocationID, "/new/nested/nested1")

		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{nestedDir})
		renameOp := sdkClient.AddRenameOperation(t, allocationID, "/new", "rename")
		sdkClient.MultiOperation(t, allocationID, []sdk.OperationRequest{renameOp})

		listResult := sdkClient.GetFileList(t, allocationID, "/rename/")
		require.Equal(t, 1, len(listResult.Children), "files count mismatch expected %v actual %v", 1, len(listResult.Children))
		listResult = sdkClient.GetFileList(t, allocationID, "/rename/nested")
		require.Equal(t, 1, len(listResult.Children), "files count mismatch expected %v actual %v", 1, len(listResult.Children))
	})
}

func randName() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")
	b := make([]rune, 10)

	for i := range b {
		ind, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letterRunes))))
		b[i] = letterRunes[ind.Int64()]
	}
	return string(b)
}

func createAllocationAndPerformMultiOperation(t *test.SystemTest, allocSize int64, filesCount, expectedFilesCount int, fileWithFormats bool, fileSizes []int64, secondaryOperation string) {
	apiClient.ExecuteFaucet(t, sdkWallet, client.TxSuccessfulStatus)
	blobberRequirements := model.DefaultBlobberRequirements(sdkWallet.Id, sdkWallet.PublicKey)
	if allocSize != 0 {
		blobberRequirements.Size = allocSize
	}
	allocationBlobbers := apiClient.GetAllocationBlobbers(t, sdkWallet, &blobberRequirements, client.HttpOkStatus)
	allocationID := apiClient.CreateAllocation(t, sdkWallet, allocationBlobbers, client.TxSuccessfulStatus)

	ops := make([]sdk.OperationRequest, 0, 10)

	for i := 0; i < filesCount; i++ {
		var fileSize int64

		if len(fileSizes) > 0 {
			lenFileSizes := len(fileSizes)
			fileSize = fileSizes[i%lenFileSizes]
		} else {
			fileSize = 1 * KB
		}

		format := ""
		if fileWithFormats {
			format = fileExtensions[i]
		}

		op := sdkClient.AddUploadOperation(t, format, fileSize)
		ops = append(ops, op)
	}
	start := time.Now()
	sdkClient.MultiOperation(t, allocationID, ops)
	end := time.Since(start)
	t.Logf("Multi upload operations took %v", end)

	if secondaryOperation != "" {
		newOps := make([]sdk.OperationRequest, 0, 10)

		if secondaryOperation == "delete" {
			for i := 0; i < 10; i++ {
				op := sdkClient.AddDeleteOperation(t, allocationID, ops[i].FileMeta.RemotePath)
				newOps = append(newOps, op)
			}
		} else if secondaryOperation == "update" {
			for i := 0; i < 10; i++ {
				op := sdkClient.AddUpdateOperation(t, allocationID, ops[i].FileMeta.RemotePath, ops[i].FileMeta.RemoteName)
				newOps = append(newOps, op)
			}
		} else if secondaryOperation == "rename" {
			for i := 0; i < 10; i++ {
				op := sdkClient.AddRenameOperation(t, allocationID, ops[i].FileMeta.RemotePath, randName())
				newOps = append(newOps, op)
			}
		}

		start := time.Now()
		sdkClient.MultiOperation(t, allocationID, newOps)
		end := time.Since(start)
		t.Logf("Multi "+secondaryOperation+"operations took %v", end)
	}

	listResult := sdkClient.GetFileList(t, allocationID, "/")
	require.Equal(t, expectedFilesCount, len(listResult.Children), "files count mismatch expected %v actual %v", 10, len(listResult.Children))
}
