package utils

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// readConfiguredSharderURLs reads sharder URLs from the zbox_config.yaml.
// The MagicBlock (used by ls-sharders) may only contain stale sharders,
// so we also check the explicitly configured sharders.
func readConfiguredSharderURLs(cfgPath string) []string {
	data, err := os.ReadFile("./config/" + cfgPath)
	if err != nil {
		return nil
	}
	var urls []string
	inSharders := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "sharders:" {
			inSharders = true
			continue
		}
		if inSharders {
			if strings.HasPrefix(trimmed, "- http") {
				url := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
				urls = append(urls, url)
			} else if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "-") {
				inSharders = false
			}
		}
	}
	return urls
}

// queryBestSharderURL queries a list of sharder base URLs and returns the one with the highest LFB round.
func queryBestSharderURL(urls []string, bestRound int64, bestURL string) (int64, string) {
	type lfbResp struct {
		Round int64 `json:"round"`
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	for _, url := range urls {
		resp, httpErr := httpClient.Get(url + "/v1/block/get/latest_finalized")
		if httpErr != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		var lfb lfbResp
		if decErr := json.NewDecoder(resp.Body).Decode(&lfb); decErr == nil && lfb.Round > bestRound {
			bestRound = lfb.Round
			bestURL = url
		}
		resp.Body.Close()
	}
	return bestRound, bestURL
}

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

	// Scan for "MagicBlock Sharders" line (zwallet may print wallet creation messages before it)
	found := false
	for index, line := range output {
		if line == "MagicBlock Sharders" {
			found = true
			output = output[index:]
			break
		}
	}
	require.True(t, found, "MagicBlock Sharders not found in getSharders output: %v", strings.Join(output, "\n"))
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	// Build list of sharder URLs to check: MB sharders + configured sharders (config may have
	// sharders not in the MagicBlock, e.g. after a partial view change or DKG issue)
	var mbURLs []string
	for _, sharder := range sharders {
		mbURLs = append(mbURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	configuredURLs := readConfiguredSharderURLs(configPath)
	// Merge: add configured URLs not already in MB list
	seen := make(map[string]bool)
	for _, u := range mbURLs {
		seen[u] = true
	}
	for _, u := range configuredURLs {
		if !seen[u] {
			mbURLs = append(mbURLs, u)
			seen[u] = true
		}
	}

	bestRound, bestURL := queryBestSharderURL(mbURLs, 0, "")
	if bestURL != "" {
		t.Logf("Using healthy sharder with round %d: %s", bestRound, bestURL)
		return bestURL
	}

	// Fallback to first sharder if none are healthy
	t.Logf("Warning: no healthy sharder found, falling back to first sharder")
	sharder := sharders[reflect.ValueOf(sharders).MapKeys()[0].String()]
	return getNodeBaseURL(sharder.Host, sharder.Port)
}

// GetSharderUrlSafe returns the sharder URL or empty string if unavailable.
// Unlike GetSharderUrl, this does not call require/assert, making it safe
// for use in skip-check contexts where we don't want to fail the test.
func GetSharderUrlSafe(t *test.SystemTest) string {
	t.Logf("getting sharder url (safe)...")
	output, err := getSharders(t, configPath)
	if err != nil || len(output) <= 1 {
		t.Logf("Warning: could not get sharders: %v", err)
		return ""
	}

	found := false
	for index, line := range output {
		if line == "MagicBlock Sharders" {
			found = true
			output = output[index:]
			break
		}
	}
	if !found || len(output) < 2 {
		t.Logf("Warning: MagicBlock Sharders not found in output")
		return ""
	}

	var sharders map[string]climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	if err != nil || len(sharders) == 0 {
		t.Logf("Warning: could not parse sharders: %v", err)
		return ""
	}

	// Build list of sharder URLs: MB sharders + configured sharders
	var mbURLs []string
	for _, sharder := range sharders {
		mbURLs = append(mbURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	configuredURLs := readConfiguredSharderURLs(configPath)
	seen := make(map[string]bool)
	for _, u := range mbURLs {
		seen[u] = true
	}
	for _, u := range configuredURLs {
		if !seen[u] {
			mbURLs = append(mbURLs, u)
			seen[u] = true
		}
	}

	_, bestURL := queryBestSharderURL(mbURLs, 0, "")
	if bestURL != "" {
		return bestURL
	}

	// Fallback to first sharder
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

func GetSortedSharderIds(t *test.SystemTest, sharderBaseURL string) []string {
	return getSortedNodeIds(t, "getSharderList", sharderBaseURL)
}
