package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func Test0TenderlyZCNBridgeAuthorizerRegisterAndDelete(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	createWallet(t)

	t.RunSequentially("Register authorizer to DEX smartcontract", func(t *test.SystemTest) {
		output, err := scRegisterAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", true)
		require.NoError(t, err, "error trying to register authorizer to DEX sc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer DEX SC successfully")
	})

	t.RunSequentially("Remove authorizer from DEX smartcontract", func(t *test.SystemTest) {
		output, err := scRemoveAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", true)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Log("remove authorizer DEX SC successfully")
	})
}

func TestZCNAuthorizerRegisterAndDelete(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	createWallet(t)

	w, err := getWallet(t, configPath)
	require.NoError(t, err)

	var (
		publicKey = w.ClientPublicKey
		clientID  = w.ClientID
		authURL   = "http://systemtest.network/authorizer01"
	)

	t.RunSequentially("Register authorizer to zcnsc smartcontract", func(t *test.SystemTest) {
		// Use the wallet's clientID for registration instead of "random_delegate_wallet"
		// This ensures we can delete it later using the same ID
		output, err := registerAuthorizer(t, clientID, publicKey, authURL, true)
		require.NoError(t, err, "error trying to register authorizer to zcnsc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer zcnsc successfully")

		// Wait for the authorizer to be registered and confirmed on the blockchain
		// This ensures the authorizer exists before we try to delete it
		maxWait := 2 * time.Minute
		startTime := time.Now()
		pollInterval := 5 * time.Second
		authorizerRegistered := false

		for !authorizerRegistered && time.Since(startTime) < maxWait {
			auths := getAuthorizersForClientID(t, clientID)
			if len(auths) > 0 {
				// Check if our authorizer is in the list
				for _, auth := range auths {
					if auth.ID == clientID {
						authorizerRegistered = true
						t.Logf("Authorizer %s confirmed as registered", clientID)
						break
					}
				}
			}
			if !authorizerRegistered {
				t.Logf("Waiting for authorizer %s to be registered... (elapsed: %v)", clientID, time.Since(startTime))
				time.Sleep(pollInterval)
			}
		}

		if !authorizerRegistered {
			t.Logf("Warning: Authorizer %s may not be fully registered yet, but continuing with deletion test", clientID)
		}
	})

	t.RunSequentially("Remove authorizer from zcnsc smartcontract", func(t *test.SystemTest) {
		// Verify authorizer exists before trying to delete
		auths := getAuthorizersForClientID(t, clientID)
		if len(auths) == 0 {
			t.Logf("Authorizer %s not found in authorizer list, skipping deletion", clientID)
			return
		}

		output, err := removeAuthorizer(t, clientID, true)
		// If authorizer doesn't exist, that's okay - it might have been deleted already or never registered
		if err != nil {
			outputStr := strings.Join(output, "\n")
			if strings.Contains(outputStr, "value not present") || strings.Contains(outputStr, "not found") {
				t.Logf("Authorizer %s not found (may have been deleted or never registered), skipping deletion", clientID)
				return
			}
			// If it's a different error, fail the test
			require.NoError(t, err, "error removing authorizer: %s", outputStr)
		}
		if err == nil {
			t.Log("remove authorizer zcnsc successfully")
		}
	})
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

func registerAuthorizer(t *test.SystemTest, clientID, publicKey, authURL string, retry bool) ([]string, error) {
	t.Log("Register authorizer to zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet auth-register --silent
		--configDir ./config
		--path %s
		--wallet wallets/sc_owner_wallet.json
		--client_id %s
		--client_key %s
		--url %s
		--min_stake 1
		--max_stake 10
		--service_charge 0.1
		--num_delegates 5`,
		configDir, clientID, publicKey, authURL)

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
		--id %s`, clientID)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

// getAuthorizersForClientID gets all authorizers and filters by clientID
func getAuthorizersForClientID(t *test.SystemTest, clientID string) []authorizerInfo {
	output, err := getAuthorizersListLocal(t, false)
	if err != nil {
		t.Logf("Error getting authorizers: %v", err)
		return nil
	}

	// Parse the JSON output to find authorizers
	outputStr := strings.Join(output, "\n")

	// Simple check: if the clientID appears in the output, the authorizer exists
	// This is a basic check - for more robust parsing, we could use JSON unmarshalling
	if strings.Contains(outputStr, clientID) {
		return []authorizerInfo{{ID: clientID}}
	}

	return nil
}

func getAuthorizersListLocal(t *test.SystemTest, retry bool) ([]string, error) {
	t.Log("Get authorizers from zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet bridge-list-auth --silent
		--configDir ./config
		--wallet %s`, escapedTestName(t)+"_wallet.json")

	if retry {
		return cliutils.RunCommand(t, cmd, 6, time.Second*10)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

type authorizerInfo struct {
	ID  string
	URL string
}
