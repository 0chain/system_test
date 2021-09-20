package cli_tests

import (
	"encoding/json"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCreateDir(t *testing.T) {

	t.Run("parallel", func(t *testing.T) {
		t.Run("create root dir", func(t *testing.T) {
			t.Parallel()
			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/rootdir")
			require.Nil(t, err, "Unexpected create dir failure", strings.Join(output, "\n"))
			require.Len(t, output, 0)

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON", err)

			wantFile := cli_model.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d", Size: 8, Hash: ""}
			require.Len(t, files, 1)
			require.Equal(t, wantFile, files[0])
		})

		t.Run("create nested dir", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/parent")
			require.Nil(t, err, "Unexpected create dir failure", strings.Join(output, "\n"))
			require.Len(t, output, 0)

			output, err = createDir(t, configPath, allocID, "/parent/child")
			require.Nil(t, err, "Unexpected create dir failure", strings.Join(output, "\n"))
			require.Len(t, output, 0)

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON", err)

			require.Len(t, files, 2)
			require.Contains(t, files, cli_model.AllocationFile{Name: "parent", Path: "/parent", Type: "d", Size: 8, Hash: ""})
			require.Contains(t, files, cli_model.AllocationFile{Name: "child", Path: "/parent/child", Type: "d", Size: 8, Hash: ""})
		})

		t.Run("create with long dir", func(t *testing.T) {

		})

		t.Run("create attempt with existing dir", func(t *testing.T) {

		})

		t.Run("create attempt with existing dir but different case", func(t *testing.T) {

		})

		t.Run("create attempt with non-existent parent dir", func(t *testing.T) {

		})

		t.Run("create attempt with dirname not a dir", func(t *testing.T) {

		})

		t.Run("create attempt with missing dirname param", func(t *testing.T) {

		})

		t.Run("create attempt with invalid allocation", func(t *testing.T) {

		})

		t.Run("create attempt with not-authorized allocation", func(t *testing.T) {

		})

		t.Run("create attempt with name identical to existing file", func(t *testing.T) {

		})
	})
}

func createDir(t *testing.T, cliConfigFilename string, allocationID string, dirname string) ([]string, error) {
	return cli_utils.RunCommand("./zbox createdir --silent --allocation " + allocationID +
		" --dirname " + dirname +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}

func listAll(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	return cli_utils.RunCommand("./zbox list-all --silent --allocation " + allocationID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
