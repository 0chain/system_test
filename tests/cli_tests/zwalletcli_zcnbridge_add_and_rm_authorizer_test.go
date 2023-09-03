package cli_tests

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestZCNBridgeAuthorizerRegisterAndDelete(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	output, err := createWallet(t, configPath)
	require.NoError(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

	t.RunSequentially("Register authorizer to DEX smartcontract", func(t *test.SystemTest) {
		output, err = scRegisterAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", false)
		require.NoError(t, err, "error trying to register authorizer to DEX sc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer DEX SC successfully")
	})

	t.RunSequentially("Remove authorizer from DEX smartcontract", func(t *test.SystemTest) {
		output, err = scRemoveAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", false)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Log("remove authorizer DEX SC successfully")
	})
}

func TestZCNAuthorizerRegisterAndDelete(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	output, err := createWallet(t, configPath)
	require.NoError(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

	w, err := getWallet(t, configPath)
	require.NoError(t, err)

	var (
		publicKey = w.ClientPublicKey
		clientID  = w.ClientID
		authURL   = "http://systemtest.network/authorizer01"
	)
	// faucet pour to sc owner wallet
	output, err = executeFaucetWithTokensForWallet(t, "wallets/sc_owner", configPath, defaultInitFaucetTokens)
	require.NoError(t, err, "Unexpected faucet execution failure", strings.Join(output, "\n"))

	t.RunSequentially("Register authorizer to zcnsc smartcontract", func(t *test.SystemTest) {
		output, err := registerAuthorizer(t, clientID, publicKey, authURL, false)
		require.NoError(t, err, "error trying to register authorizer to zcnsc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer zcnsc successfully")
	})

	t.RunSequentially("Remove authorizer from zcnsc smartcontract", func(t *test.SystemTest) {
		output, err := removeAuthorizer(t, clientID, false)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Log("remove authorizer zcnsc successfully")
	})
}

func TestReplaceAuthorizerBurnZCNAndMintWZCN(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	authsIDKeys := map[string]string{
		"d6e9b3222434faa043c683d1a939d6a0fa2818c4d56e794974d64a32005330d3": "b41d6232f11e0feefe895483688410216b3b1101e5db55044b22f0342fc18718b96b3124c9373dd116c50bd9b60512f28930a0e5771e58ecdc7d5bc2b570111a",
		"7b07c0489e2f35d7c13160f4da2866b4aa69aa4e8d2b2cd9c4fc002693dca5d7": "aa6b6a16ae362189008cd4e7b4573174460965ab8d9c18515f0142cee4d8ba0708584cfbb8074120586998157ccb808954cde6c68443f22aab0b5ca72175c79d",
		"896c171639937a647f9e91d5ba676be580f6d2b7e0d708e4fe6ea36610a13ffd": "aa894f74724dbb774deafda1de89b1d2853e1849c148c632ef7c9877338d5d129c9ccca3fe6a4581af2b07bbfb1225da4f674b1f76b49bc2187dc761896dff87",
	}

	// create local wallet and faucet
	output, err := createWallet(t, configPath)
	require.NoError(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

	// We should have 3 authorizers in total when the network is deployed.
	// Test steps:
	// - Remove one authorizer, and do the burn ZCN and mint WZCN test flow.
	// Expected result: The test should pass.
	//
	// - Remove another authorizer, and add back the first removed authorizer, do the burn and mint test flow.
	// Expect result: The test should pass.
	//
	// Ideally we should check the outputs to see the mint requests burn tickets to the right authorizers.
	// but that's trivial and we have done the manual test to confirm the correct authorizers are used.
	auths := getAuthorizers(t, false)

	require.Len(t, auths, 3, "There should be 3 authorizers in the network")

	// Remove 1 authorizers from zcnsc smartcontract
	removeAuth := auths[0]
	output, err = removeAuthorizer(t, removeAuth.ID, false)
	require.NoError(t, err, "Unexpected remove authorizer failure", strings.Join(output, "\n"))
	t.Logf("remove authorizer: %s zcnsc successfully", removeAuth.ID)

	// confirm burn zcn and mint wzcn could work with the existing 2 authorizers
	burnZCNMintWZCN(t)

	// remove another authorizer
	auth2 := auths[1]
	output, err = removeAuthorizer(t, auth2.ID, false)
	require.NoError(t, err, "Unexpected remove authorizer failure", strings.Join(output, "\n"))
	t.Logf("remove authorizer: %s zcnsc successfully", auth2.ID)

	// add back the first removed authorizer
	addAuth := auths[0]
	output, err = registerAuthorizer(t, addAuth.ID, authsIDKeys[addAuth.ID], addAuth.URL, false)
	require.NoError(t, err, "Unexpected register authorizer failure", strings.Join(output, "\n"))

	// wait until the new one is registered
	var (
		// 6 * 30 seconds = 2 minute, the authorizer send health check every 60 seconds
		maxRetry = 6
		reged    bool
	)
	for i := 0; i < maxRetry; i++ {
		newAuths := getAuthorizers(t, false)
		if len(newAuths) == 2 {
			reged = true
			break
		}
		t.Logf("retry list authorizers after 10 seconds, current num: %d", len(newAuths))
		time.Sleep(30 * time.Second)
	}

	require.True(t, reged, "The new authorizer is not registered")

	// confirm burn zcn and mint wzcn could work
	burnZCNMintWZCN(t)

	// add back the second removed authorizer
	addAuth2 := auths[1]
	output, err = registerAuthorizer(t, addAuth2.ID, authsIDKeys[addAuth2.ID], addAuth2.URL, false)
	require.NoError(t, err, "Unexpected register authorizer failure", strings.Join(output, "\n"))
}

func burnZCNMintWZCN(t *test.SystemTest) {
	// Burn zcn
	output, err := burnZCN(t, "1", false)
	require.NoError(t, err, "Unexpected burn zcn failure", strings.Join(output, "\n"))
	t.Log("burn output:", strings.Join(output, "\n"))

	// Mint wzcn
	output, err = mintWZCN(t, false)
	require.NoError(t, err, "Unexpected mint wzcn failure", strings.Join(output, "\n"))
	t.Log("mint output:", strings.Join(output, "\n"))
}

func burnZCN(t *test.SystemTest, amount string, retry bool) ([]string, error) {
	t.Log("Burn zcn ...")
	cmd := fmt.Sprintf(`
		./zwallet bridge-burn-zcn --silent
		--configDir ./config
		--path %s
		--wallet %s
		--token %s`,
		configDir,
		escapedTestName(t)+"_wallet.json",
		amount)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func mintWZCN(t *test.SystemTest, retry bool) ([]string, error) {
	t.Log("Mint wzcn ...")
	cmd := fmt.Sprintf(`
		./zwallet bridge-mint-wzcn --silent
		--configDir ./config
		--path %s
		--wallet %s`,
		configDir,
		escapedTestName(t)+"_wallet.json")

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

type authorizerNode struct {
	ID  string
	URL string
}

func getAuthorizers(t *test.SystemTest, retry bool) []authorizerNode {
	// get all authorizers
	output, err := getAuthorizersCmd(t, false)
	require.NoError(t, err)

	// Define the regular expression to match the line with "id": "the authorizer id"
	re := regexp.MustCompile(`"id"\s*:\s*"(.*?)"|"url"\s*:\s*"(.*?)"`)

	// Find all matches
	matches := re.FindAllStringSubmatch(strings.Join(output, "\n"), -1)

	// Extract the URLs from the matches
	ids := make([]string, 0, len(matches))
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if match[1] != "" {
			ids = append(ids, match[1])
		} else {
			urls = append(urls, match[2])
		}
	}

	auths := make([]authorizerNode, len(ids))
	for i := 0; i < len(ids); i++ {
		auths[i] = authorizerNode{
			ID:  ids[i],
			URL: urls[i],
		}
	}
	return auths
}

func getAuthorizersCmd(t *test.SystemTest, retry bool) ([]string, error) {
	t.Log("Get authorizers from zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet bridge-list-auth --silent
		--configDir ./config
		--wallet %s`, escapedTestName(t)+"_wallet.json")

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func scRegisterAuthorizer(t *test.SystemTest, authAddress string, retry bool) ([]string, error) {
	t.Logf("Register authorizer to SC...")
	cmd := fmt.Sprintf(
		"./zwallet auth-sc-register "+
			"--ethereum_address=%s "+
			"--silent "+
			"--path %s "+
			"--configDir ./config "+
			"--wallet %s",
		authAddress,
		configDir,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func scRemoveAuthorizer(t *test.SystemTest, authAddress string, retry bool) ([]string, error) {
	t.Logf("Remove authorizer to SC ...")
	cmd := fmt.Sprintf(
		"./zwallet auth-sc-delete --ethereum_address=%s "+
			"--silent "+
			"--configDir ./config "+
			"--wallet %s",
		authAddress,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func registerAuthorizer(t *test.SystemTest, clientID, publicKey, authURL string, retry bool) ([]string, error) {
	t.Log("Register authorizer to zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet auth-register --silent
		--configDir ./config
		--wallet wallets/sc_owner_wallet.json
		--client_id %s
		--client_key %s
		--url %s
		--min_stake 1
		--max_stake 10
		--service_charge 0.1
		--num_delegates 5`,
		clientID, publicKey, authURL)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func removeAuthorizer(t *test.SystemTest, clientID string, retry bool) ([]string, error) {
	t.Log("Remove authorizer from zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet bridge-auth-delete --silent
		--configDir ./config
		--wallet wallets/sc_owner_wallet.json
		--id %s`,
		clientID)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
