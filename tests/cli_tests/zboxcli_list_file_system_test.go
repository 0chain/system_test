package cli_tests

import (
	"encoding/json"
	"fmt"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pretty(data interface{}) {
	bts, _ := json.MarshalIndent(data, "", "\t")
	fmt.Println(string(bts))
}

func TestListFileSystem(t *testing.T) {

	t.Run("No Parameter Should Fail", func(t *testing.T) {
		output, err := listFilesInAllocation(t, configPath, "")
		require.NotNil(t, err,
			"List files with no parameter failed due to error", err,
			strings.Join(output, "\n"))

		require.Equal(t, "Error: remotepath / authticket flag is missing", output[len(output)-1])
	})

	t.Run("No Files in Allocation Should Work", func(t *testing.T) {
		allocationID := setupAllocation(t, configPath)

		param := createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"json":       "",
		})

		output, err := listFilesInAllocation(t, configPath, param)
		require.Nil(t, err,
			"List files with no files in allocation failed due to error", err,
			strings.Join(output, "\n"))

		require.Equal(t, 1, len(output))
		require.Equal(t, "null", output[0])
	})

	t.Run("List Files in Root Directory Should Work", func(t *testing.T) {

		allocationID := setupAllocation(t, configPath)

		// First Upload a file to the root directory
		filename := generateTestFile(t)
		filesize := int64(10)

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": "/",
		})

		// Then list the files and check
		listParam := createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": "/",
		})

		output, err := listFilesInAllocation(t, configPath, listParam)
		require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))

		require.Equal(t, 1, len(output))

		var listResults []cli_model.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed due to error ", err)

		require.Equal(t, 1, len(listResults))
		result := listResults[0]

		require.Equal(t, filepath.Base(filename), result.Name)
		require.Equal(t, "/"+filepath.Base(filename), result.Path)
		require.Equal(t, filesize, result.ActualSize)
		require.Equal(t, "f", result.Type)
	})

	t.Run("List Files in a Directory Should Work", func(t *testing.T) {

		allocationID := setupAllocation(t, configPath)

		// First Upload a file to the a directory
		filename := generateTestFile(t)
		filesize := int64(2)
		remotepath := "/test_file/"

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
		})

		// Then list the files and check
		listParam := createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath,
		})

		output, err := listFilesInAllocation(t, configPath, listParam)
		require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))

		require.Equal(t, 1, len(output))

		var listResults []cli_model.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed due to error ", err)

		require.Equal(t, 1, len(listResults))
		result := listResults[0]

		require.Equal(t, filepath.Base(filename), result.Name)
		require.Equal(t, remotepath+filepath.Base(filename), result.Path)
		require.Equal(t, filesize, result.ActualSize)
		require.Equal(t, "f", result.Type)
	})

	t.Run("List Files in Nested Directory Should Work", func(t *testing.T) {

		allocationID := setupAllocation(t, configPath)

		// First Upload a file to the a directory
		filename := generateTestFile(t)
		filesize := int64(2)
		remotepath := "/nested/test_file/"

		err := createFileWithSize(filename, filesize)
		require.Nil(t, err)

		// Upload parameters
		uploadWithParam(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  filename,
			"remotepath": remotepath + filepath.Base(filename),
		})

		// Then list the files and check
		listParam := createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"remotepath": remotepath,
		})

		output, err := listFilesInAllocation(t, configPath, listParam)
		require.Nil(t, err, "List file failed due to error ", err, strings.Join(output, "\n"))

		require.Equal(t, 1, len(output))

		var listResults []cli_model.ListFileResult
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&listResults)
		require.Nil(t, err, "Decoding list results failed due to error ", err)

		require.Equal(t, 1, len(listResults))
		result := listResults[0]

		require.Equal(t, filepath.Base(filename), result.Name)
		require.Equal(t, remotepath+filepath.Base(filename), result.Path)
		require.Equal(t, filesize, result.ActualSize)
		require.Equal(t, "f", result.Type)
	})
}

func createFileWithSize(name string, size int64) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	if err := f.Truncate(size); err != nil {
		return err
	}

	return nil
}

func generateTestFile(t *testing.T) string {
	path, err := filepath.Abs("config")
	require.Nil(t, err)
	return fmt.Sprintf("%s/%s_test.txt", path, escapedTestName(t))
}

func uploadWithParam(t *testing.T, cliConfigFilename string, param map[string]interface{}) {

	filename, ok := param["localpath"].(string)
	require.True(t, ok)

	output, err := uploadFileInAllocation(t, cliConfigFilename, createParams(param))
	require.Nil(t, err, "Upload file failed due to error ", err, strings.Join(output, "\n"))

	require.Equal(t, 2, len(output))

	expected := fmt.Sprintf(
		"Status completed callback. Type = application/octet-stream. Name = %s",
		filepath.Base(filename),
	)
	require.Equal(t, expected, output[1])
}

func uploadFileInAllocation(t *testing.T, cliConfigFilename string, param string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox upload %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}

func listFilesInAllocation(t *testing.T, cliConfigFilename string, param string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zbox list %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)
	return cli_utils.RunCommand(cmd)
}
