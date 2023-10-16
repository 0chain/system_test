package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileCopy(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("copy file to existing directory")

	t.Parallel()
	t.Run("copy directory to another directry should work", func(t *test.SystemTest) {
		allocSize := int64(2048000)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/child2"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// copy file
		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/child",
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("/child"+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 5)

		// check if expected file has been copied. both files should be there
		foundAtSource := false
		foundAtDest := false
		destfilepath := `/child2` + remotePath
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destfilepath {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.GreaterOrEqual(t, f.Size, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy directory to another directry with no existing file should work", func(t *test.SystemTest) {
		allocSize := int64(2048000)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		dirname := "/child1"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		destpath := "/child2"
		output, err = createDir(t, configPath, allocationID, destpath, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, destpath+" directory created", output[0])

		// copy file

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(dirname+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 3)

		// check if expected file has been copied. both files should be there
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == dirname {
				foundAtSource = true
				require.Equal(t, "child1", f.Name, strings.Join(output, "\n"))
				require.Equal(t, "d", f.Type, strings.Join(output, "\n"))
			}
			if f.Path == filepath.Join(destpath, dirname) {
				foundAtDest = true
				require.Equal(t, "child1", f.Name, strings.Join(output, "\n"))
				require.Equal(t, "d", f.Type, strings.Join(output, "\n"))
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy directory to another directry with multiple existing file should work", func(t *test.SystemTest) {
		allocSize := int64(2048000)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		dirname := "/child1"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		destpath := "/child2"
		output, err = createDir(t, configPath, allocationID, destpath, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, destpath+" directory created", output[0])

		// Upload file
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename1 := filepath.Base(file)
		remotefilePath1 := "/child1/" + filename1

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotefilePath1,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		file = generateRandomTestFileName(t)
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename2 := filepath.Base(file)
		remotefilePath2 := "/child1/" + filename2

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotefilePath2,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected = fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])
		// copy file

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(dirname+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 7)

		// check if expected file has been copied. both files should be there
		foundfile1AtSource := false
		foundfile2AtSource := false
		foundDirAtSource := false
		foundDirAtDest := false
		foundfile1AtDest := false
		foundfile2AtDest := false
		//nolint: gocritic
		for _, f := range files {
			if f.Path == remotefilePath1 {
				foundfile1AtSource = true
				require.Equal(t, filename1, f.Name, strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			} else if f.Path == remotefilePath2 {
				foundfile2AtSource = true
				require.Equal(t, filename2, f.Name, strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			} else if f.Path == "/child2"+remotefilePath1 {
				foundfile1AtDest = true
				require.Equal(t, filename1, f.Name, strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			} else if f.Path == "/child2"+remotefilePath2 {
				foundfile2AtDest = true
				require.Equal(t, filename2, f.Name, strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			} else if f.Path == dirname {
				foundDirAtSource = true
				require.Equal(t, "child1", f.Name, strings.Join(output, "\n"))
				require.Equal(t, "d", f.Type, strings.Join(output, "\n"))
			} else if f.Path == filepath.Join(destpath, "/", dirname) {
				foundDirAtDest = true
				require.Equal(t, "child1", f.Name, strings.Join(output, "\n"))
				require.Equal(t, "d", f.Type, strings.Join(output, "\n"))
			}
		}
		require.True(t, foundDirAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundDirAtDest, "file not found at destination: ", strings.Join(output, "\n"))
		require.True(t, foundfile1AtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundfile1AtDest, "file not found at destination: ", strings.Join(output, "\n"))
		require.True(t, foundfile2AtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundfile2AtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy file to existing directory", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// copy file
		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 3)

		// check if expected file has been copied. both files should be there
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.RunWithTimeout("Copy file concurrently to existing directory, should work", 6*time.Minute, func(t *test.SystemTest) { // todo: way too slow
		const allocSize int64 = 2048
		const fileSize int64 = 256

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		var fileNames [2]string
		var remoteFilePaths, destFilePaths []string

		const remotePathPrefix = "/"
		const destPathPrefix = "/new"

		var outputList [2][]string
		var errorList [2]error
		var wg sync.WaitGroup

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(currentIndex int) {
				defer wg.Done()

				fileName := filepath.Base(generateFileAndUpload(t, allocationID, remotePathPrefix, fileSize))
				fileNames[currentIndex] = fileName

				remoteFilePath := filepath.Join(remotePathPrefix, fileName)
				remoteFilePaths = append(remoteFilePaths, remoteFilePath)

				destFilePath := filepath.Join(destPathPrefix, fileName)
				destFilePaths = append(destFilePaths, destFilePath)

				op, err := copyFile(t, configPath, map[string]interface{}{
					"allocation": allocationID,
					"remotepath": remoteFilePath,
					"destpath":   destPathPrefix,
				}, true)

				errorList[currentIndex] = err
				outputList[currentIndex] = op
			}(i)
		}

		wg.Wait()

		const expectedPattern = "%s copied"

		for i := 0; i < 2; i++ {
			require.Nil(t, errorList[i], strings.Join(outputList[i], "\n"))
			require.Len(t, outputList[i], 1, strings.Join(outputList[i], "\n"))

			require.Equal(t, fmt.Sprintf(expectedPattern, fileNames[i]), filepath.Base(outputList[i][0]), "Output is not appropriate")
		}

		output, err := listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 5)

		var foundAtSource, foundAtDest int
		for _, f := range files {
			if _, ok := cliutils.Contains(remoteFilePaths, f.Path); ok {
				foundAtSource++
			}

			if _, ok := cliutils.Contains(destFilePaths, f.Path); ok {
				foundAtDest++

				_, ok = cliutils.Contains(fileNames[:], f.Name)
				require.True(t, ok, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.Equal(t, 2, foundAtSource, "file is found at source: ", strings.Join(output, "\n"))
		require.Equal(t, 2, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy file to non-existing directory should work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/child/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 3)

		// check if expected file has been copied. both files should be there
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy file to same directory should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Copy failed")

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 1)

		// check if file is still there
		found := false
		for _, f := range files {
			if f.Path == remotePath { // nolint:gocritic // this is better than inverted if cond
				found = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, found, "file not found: ", strings.Join(output, "\n"))
	})

	t.Run("copy file to dir with existing children should work", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/testing_path/copy_here/children/" + filename
		destpath := "/testing_path/copy_here"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotePath+" copied"), output[0])

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		assert.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		assert.Len(t, files, 5)

		// check if file is still there
		found := false
		for _, f := range files {
			if f.Path == remotePath { // nolint:gocritic // this is better than inverted if cond
				found = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, found, "file not found: ", strings.Join(output, "\n"))
	})

	t.Run("copy file to another directory with existing file with same name should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		existingFileInDest := generateRandomTestFileName(t)
		err = createFileWithSize(existingFileInDest, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/target/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		// upload file to another directory with same name.
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": destpath + filename,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected = fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Copy failed")

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 3)

		// check if both existing files are there
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destpath+filename {
				foundAtDest = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.True(t, foundAtDest, "file not found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy non-existing file should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		// try to copy a file
		output, err := copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/child/nonexisting.txt",
			"destpath":   "/",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Copy failed")
	})

	t.Run("copy file from someone else's allocation should fail", func(t *test.SystemTest) {
		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err := createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err = createFileWithSize(file, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/child/" + filename
		destpath := "/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": allocSize,
		})

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(file),
		)
		require.Equal(t, expected, output[1])

		output, err = copyFileForWallet(t, configPath, nonAllocOwnerWallet, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "Copy failed")

		// list-all
		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)
		require.Len(t, files, 2)

		// check if file was not copied
		foundAtSource := false
		foundAtDest := false
		for _, f := range files {
			if f.Path == remotePath {
				foundAtSource = true
				require.Equal(t, filename, f.Name, strings.Join(output, "\n"))
				require.Equal(t, f.ActualSize, int(fileSize), strings.Join(output, "\n"))
				require.Equal(t, "f", f.Type, strings.Join(output, "\n"))
				require.NotEmpty(t, f.Hash)
			}
			if f.Path == destpath+filename {
				foundAtDest = true
			}
		}
		require.True(t, foundAtSource, "file not found at source: ", strings.Join(output, "\n"))
		require.False(t, foundAtDest, "file is found at destination: ", strings.Join(output, "\n"))
	})

	t.Run("copy file with no allocation param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on copy
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = copyFile(t, configPath, map[string]interface{}{
			"remotepath": "/abc.txt",
			"destpath":   "/",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("copy file with no remotepath param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on copy
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"destpath":   "/",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: remotepath flag is missing", output[0])
	})

	t.Run("copy file with no destpath param should fail", func(t *test.SystemTest) {
		// unused wallet, just added to avoid having the creating new wallet outputs on copy
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": "abcdef",
			"remotepath": "/abc.txt",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: destpath flag is missing", output[0])
	})

	t.Run("copy file with allocation copy file options forbidden should fail", func(t *test.SystemTest) {
		allocSize := int64(2048)
		fileSize := int64(256)

		file := generateRandomTestFileName(t)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		existingFileInDest := generateRandomTestFileName(t)
		err = createFileWithSize(existingFileInDest, fileSize)
		require.Nil(t, err)

		filename := filepath.Base(file)
		remotePath := "/" + filename
		destpath := "/target/"

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":        allocSize,
			"forbid_copy": nil,
		})

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  file,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"destpath":   destpath,
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Contains(t, output[0], "this options for this file is not permitted for this allocation")

		output, err = listFilesInAllocation(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": destpath,
		}), false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Contains(t, strings.Join(output, "\n"), "Invalid path record not found")
	})

	t.RunWithTimeout("File copy - Users should be charged for copying a file ", 5*time.Minute, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))
		t.Logf("Wallet created: %s", output[0])

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))
		t.Logf("Faucet executed: %s", output[0])

		// Lock 0.5 token for allocation
		allocParams := createParams(map[string]interface{}{
			"lock": "0.5",
			"size": 4 * MB,
		})
		output, err = createNewAllocation(t, configPath, allocParams)
		require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))
		t.Logf("Allocation created: %s", output[0])

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("Allocation created: ([a-f0-9]{64})"), output[0], "Allocation creation output did not match expected")
		allocationID := strings.Fields(output[0])[2]
		t.Logf("Allocation ID: %s", allocationID)

		initialAllocation := getAllocation(t, allocationID)
		t.Logf("Initial allocation: %+v", initialAllocation)

		fileSize := int64(math.Floor(1 * MB))

		// Upload 1 MB file
		localpath := uploadRandomlyGeneratedFile(t, allocationID, "/", fileSize)
		output, _ = getUploadCostInUnit(t, configPath, allocationID, localpath)
		expectedUploadCostInZCN, err := strconv.ParseFloat(strings.Fields(output[0])[0], 64)
		require.Nil(t, err, "Cost couldn't be parsed to float", strings.Join(output, "\n"))
		t.Logf("Upload cost: %v", expectedUploadCostInZCN)
		unit := strings.Fields(output[0])[1]
		t.Logf("Upload cost unit: %v", unit)
		expectedUploadCostInZCN = unitToZCN(expectedUploadCostInZCN, unit)
		t.Logf("Upload cost in ZCN: %v", expectedUploadCostInZCN)

		time.Sleep(30 * time.Second)

		allocAfterUpload := getAllocation(t, allocationID)
		require.Equal(t, initialAllocation.WritePool-allocAfterUpload.WritePool, allocAfterUpload.MovedToChallenge)
		require.InEpsilon(t, expectedUploadCostInZCN, intToZCN(allocAfterUpload.MovedToChallenge), 0.05, "Upload cost is not as expected %v != %v", expectedUploadCostInZCN, intToZCN(allocAfterUpload.MovedToChallenge))

		remotepath := "/" + filepath.Base(localpath)
		// copy file
		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"destpath":   "/newdir/",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf(remotepath+" copied"), output[0])

		cliutils.Wait(t, 30*time.Second)

		finalAllocation := getAllocation(t, allocationID)
		finalAllocationJSON, err := json.Marshal(allocAfterUpload)
		require.Nil(t, err, "Failed to marshal allocation", strings.Join(output, "\n"))
		t.Log("finalAllocationJSON: ", string(finalAllocationJSON))

		actualCost := finalAllocation.MovedToChallenge - allocAfterUpload.MovedToChallenge

		t.Logf("Actual cost: %v", actualCost)
		t.Log("expectedUploadCostInZCN : ", expectedUploadCostInZCN, " actualCost : ", intToZCN(actualCost))

		require.InEpsilon(t, expectedUploadCostInZCN, intToZCN(actualCost), 0.05, "Copy file cost is not as expected")

		createAllocationTestTeardown(t, allocationID)
	})
}

func copyFile(t *test.SystemTest, cliConfigFilename string, param map[string]interface{}, retry bool) ([]string, error) {
	return copyFileForWallet(t, cliConfigFilename, escapedTestName(t), param, retry)
}

func copyFileForWallet(t *test.SystemTest, cliConfigFilename, wallet string, param map[string]interface{}, retry bool) ([]string, error) {
	t.Logf("Copying file...")
	p := createParams(param)
	cmd := fmt.Sprintf(
		"./zbox copy %s --silent --wallet %s --configDir ./config --config %s",
		p,
		wallet+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*20)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
