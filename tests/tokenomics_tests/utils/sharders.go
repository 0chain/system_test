package utils

import (
	"encoding/json" //nolint:goimports
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"reflect"
	"strings"
)

func getShardersList(t *test.SystemTest) map[string]climodel.Sharder {
	return getShardersListForWallet(t, EscapedTestName(t))
}

func getShardersListForWallet(t *test.SystemTest, wallet string) map[string]climodel.Sharder { // Get sharder list.
	output, err := getShardersForWallet(t, configPath, wallet)
	found := false
	for index, line := range output {
		if line == "MagicBlock Sharders" {
			found = true
			output = output[index:]
			break
		}
	}
	require.True(t, found, "MagicBlock Sharders not found in getShardersForWallet output")
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 0)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	return sharders
}

func GetSharderUrl(t *test.SystemTest) string {
	t.Logf("getting sharder url...")
	// Get sharder list.
	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]

	return getNodeBaseURL(sharder.Host, sharder.Port)
}

func getSharders(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return getShardersForWallet(t, cliConfigFilename, EscapedTestName(t))
}

func getShardersForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	t.Logf("list sharder nodes...")
	return cliutil.RunCommandWithRawOutput("./zwallet ls-sharders --active --json --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
