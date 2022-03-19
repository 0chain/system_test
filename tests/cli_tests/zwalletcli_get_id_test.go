package cli_tests

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestGetId(t *testing.T) {
	t.Parallel()

	t.Run("get miner id should work", func(t *testing.T) {
		t.Parallel()

		miners := getMinersList(t)
		minerUrl := fmt.Sprint("http://", miners.Nodes[0].Host, ":", miners.Nodes[0].Port)

		output, err := getId(t, configPath, minerUrl, true)
		require.Nil(t, err, "get id failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1, "Expected output length to be at least 2", strings.Join(output, "\n"))
		require.Equal(t, "URL: "+minerUrl, output[len(output)-2], strings.Join(output, "\n"))
		require.Equal(t, "ID: "+miners.Nodes[0].ID, output[len(output)-1], strings.Join(output, "\n"))
	})

	t.Run("get sharder id should work", func(t *testing.T) {
		t.Parallel()

		_, _ = registerWallet(t, configPath)

		sharders := getShardersList(t)
		sharderKey := reflect.ValueOf(sharders).MapKeys()[0].String()
		sharder := sharders[sharderKey]
		sharderUrl := fmt.Sprint("http://", sharder.Host, ":", sharder.Port)
		require.NotNil(t, sharder)

		output, err := getId(t, configPath, sharderUrl, true)
		require.Nil(t, err, "get is failed", strings.Join(output, "\n"))
		require.Greater(t, len(output), 1, "Expected output length to be at least 2", strings.Join(output, "\n"))
		require.Equal(t, "URL: "+sharderUrl, output[len(output)-2], strings.Join(output, "\n"))
		require.Equal(t, "ID: "+sharder.ID, output[len(output)-1], strings.Join(output, "\n"))
	})

	t.Run("get blobber id should not work", func(t *testing.T) {
		t.Parallel()

		_, _ = registerWallet(t, configPath)

		blobbers := getBlobbersList(t)
		blobberUrl := blobbers[0].Url

		output, err := getId(t, configPath, blobberUrl, false)
		require.NotNil(t, err, "expected get id to fail", strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "Error: ID not found", output[0], strings.Join(output, "\n"))
	})
}

func getId(t *testing.T, cliConfigFilename, url string, retry bool) ([]string, error) {
	t.Logf("getting id for [%s]...", url)
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet getid --silent --configDir ./config --url %s --config %s", url, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet getid --silent --configDir ./config --url %s --config %s", url, cliConfigFilename))
	}
}
