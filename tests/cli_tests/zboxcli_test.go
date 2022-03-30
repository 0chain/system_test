package cli_tests

import (
	"regexp"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

// CreateWallet create new wallet for tests. a wallet can be used on multiple isolated alloation tests
func CreateWallet(t *testing.T, configName, walletName string) *Wallet {
	w := &Wallet{
		ConfigName: configName,
		WalletName: walletName,
	}

	faucetTokens := 3.0
	// First create a wallet and run faucet command
	// Output:
	// 		[0]:"ZCN wallet created"
	// 		[1]:"Creating related read pool for storage smart-contract..."
	// 		[2]:"Read pool created successfully"
	// 		[3]:"Wallet registered"
	output, err := registerWalletForName(t, configPath, walletName)
	require.Nil(t, err, "registering wallet failed", strings.Join(output, "\n"))
	require.Len(t, output, 4, strings.Join(output, "\n"))
	require.Equal(t, "Read pool created successfully", output[2], strings.Join(output, "\n"))
	require.Equal(t, "Wallet registered", output[3], strings.Join(output, "\n"))

	output, err = executeFaucetWithTokensForWallet(t, walletName, configPath, faucetTokens)
	require.Nil(t, err, "faucet execution failed", strings.Join(output, "\n"))

	return w
}

// Wallet a shared walllet for non-parallel wallet tests
type Wallet struct {
	ConfigName string
	WalletName string

	Client *climodel.Wallet
}

var (
	createAllocationParams = map[string]interface{}{
		"lock":   0.5,
		"size":   10485760,
		"expire": "2h",
		"parity": 1,
		"data":   1,
	}
)

// CreateAllocation create new allocation for tests
func (w *Wallet) CreateAllocation(t *testing.T, params map[string]interface{}) *Allocation {
	a := &Allocation{}

	if params == nil {
		params = createAllocationParams
	}

	allocParam := createParams(params)

	output, err := createNewAllocationForWallet(t, w.WalletName, w.ConfigName, allocParam)

	require.Nil(t, err, "Failed to create new allocation", strings.Join(output, "\n"))

	require.Len(t, output, 1)
	matcher := regexp.MustCompile("Allocation created: ([a-f0-9]{64})")
	require.Regexp(t, matcher, output[0], "Allocation creation output did not match expected")

	allocationID := strings.Fields(output[0])[2]

	// locking tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"allocation": allocationID,
		"tokens":     0.4,
		"duration":   "1h",
	})
	output, err = readPoolLockWithWallet(t, w.WalletName, w.ConfigName, readPoolParams, true)
	require.Nil(t, err, "Tokens could not be locked", strings.Join(output, "\n"))
	require.Len(t, output, 1)
	require.Equal(t, "locked", output[0])

	w.Client, err = getWalletForName(t, w.ConfigName, w.WalletName)
	require.Nil(t, err)

	a.id = allocationID
	a.params = params

	return a
}

// Wallet a shared allocation for non-parallel allocation tests
type Allocation struct {
	Wallet *Wallet

	id     string
	params map[string]interface{}
}
