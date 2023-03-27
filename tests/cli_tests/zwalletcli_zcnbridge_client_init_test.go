package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/test"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

const (
	DefaultConfigBridgeFileName = "bridge.yaml"
	DefaultConfigOwnerFileName  = "owner.yaml"
)

const (
	OptionHash             = "hash"          // OptionHash hash passed to cmd
	OptionAmount           = "amount"        // OptionAmount amount passed to cmd
	OptionRetries          = "retries"       // OptionRetries retries
	OptionConfigFolder     = "path"          // OptionConfigFolder config folder
	OptionChainConfigFile  = "chain_config"  // OptionChainConfigFile sdk config filename
	OptionBridgeConfigFile = "bridge_config" // OptionBridgeConfigFile bridge config filename
	OptionOwnerConfigFile  = "owner_config"  // OptionOwnerConfigFile bridge owner config filename
	OptionMnemonic         = "mnemonic"      // OptionMnemonic bridge config filename
	OptionKeyPassword      = "password"      // OptionKeyPassword bridge config filename
)

func PrepareBridgeClient(t *test.SystemTest) error {
	_, err := prepareBridgeClientConfig(t)
	if err != nil {
		return err
	}

	_, err = prepareBridgeClientWallet(t)
	if err != nil {
		return err
	}

	return nil
}

// Tests prerequisites
func prepareBridgeClientConfig(t *test.SystemTest) ([]string, error) {
	return runCreateBridgeClientTestConfig(
		t,
		"\"02289b9\"",
		"0xD8c9156e782C68EE671C09b6b92de76C97948432",
		"0x0c2aa005C6FF9F4B46Ae566D9bc61E33B482D8E6",
		"0xbD2048E2348b8Eb597D356AF23EAfAa246F88375",
		"https://goerli.infura.io/v3/773bfe30452f40f998e5d0f2f8a29888 ",
		75,
		300000,
		0,
		WithOption(OptionConfigFolder, configDir),
		WithOption(OptionBridgeConfigFile, bridgeClientConfigFile),
	)
}

// Use it to import account to the given home folder
func prepareBridgeClientWallet(t *test.SystemTest) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zwallet bridge-import-account --%s %s --%s %q --%s %s",
		OptionConfigFolder, configDir,
		OptionMnemonic, mnemonic,
		OptionKeyPassword, password,
	)

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// cmd: bridge-client-init
func TestCoreBridgeClientInit(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Init bridge client config to default path and file", func(t *test.SystemTest) {
		output, err := createDefaultClientBridgeConfig(t)

		defaultPath := filepath.Join(getZCNDir(), DefaultConfigBridgeFileName)

		require.Nil(t, err, "error trying to create an initial client bridge config", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", defaultPath), output[len(output)-1])
	})

	customPath := filepath.Join(getConfigDir(), "test")

	t.RunSequentially("Init bridge client config to custom path and default file", func(t *test.SystemTest) {
		//goland:noinspection GoUnhandledErrorResult
		defer os.RemoveAll(customPath)

		if _, err := os.Stat(customPath); os.IsNotExist(err) {
			err := os.MkdirAll(customPath, 0755)
			require.NoError(t, err)
		} else {
			err := os.RemoveAll(customPath)
			require.Error(t, err)
		}

		output, err := bridgeClientInit(t,
			"password",
			"0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947",
			"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
			"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
			"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
			0.75,
			300000,
			0,
			WithOption("path", customPath),
		)

		customPath := filepath.Join(customPath, DefaultConfigBridgeFileName)

		require.Nil(t, err, "error trying to create an initial client bridge config", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", customPath), output[len(output)-1])
	})

	t.RunSequentially("Init bridge client config to custom path and custom config file", func(t *test.SystemTest) {
		//goland:noinspection GoUnhandledErrorResult
		defer os.RemoveAll(customPath)

		if _, err := os.Stat(customPath); os.IsNotExist(err) {
			err := os.MkdirAll(customPath, 0755)
			require.NoError(t, err)
		} else {
			err := os.RemoveAll(customPath)
			require.Error(t, err)
		}

		output, err := bridgeClientInit(t,
			"password",
			"0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947",
			"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
			"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
			"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
			0.75,
			300000,
			0,
			WithOption("path", customPath),
			WithOption("bridge_config", "customName.yaml"),
		)

		customPath := filepath.Join(customPath, "customName.yaml")

		require.Nil(t, err, "error trying to create an initial client bridge config", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", customPath), output[len(output)-1])
	})
}

// cmd: bridge-owner-init
func TestCoreBridgeOwnerInit(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Run("Init bridge owner config to default path and file", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))
		output, err = bridgeOwnerInit(
			t,
			"password",
			"0x860FA46F170a87dF44D7bB867AA4a5D2813127c1",
			"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
			"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
			"0xFE20Ce9fBe514397427d20C91CB657a4478A0FFa",
			"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
			300000,
			0,
		)

		defaultPath := filepath.Join(getZCNDir(), DefaultConfigOwnerFileName)

		require.Nil(t, err, "error trying to create an initial owner config", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Equal(t, fmt.Sprintf("Owner config file was saved to %s", defaultPath), output[len(output)-1])
	})
}

// cmd: bridge-client-init
func bridgeClientInit(
	t *test.SystemTest,
	password, ethereumaddress, bridgeaddress, wzcnaddress, ethereumnodeurl string,
	consensusthreshold float64,
	gaslimit, value int64,
	opts ...*Option,
) ([]string, error) {
	t.Logf("Init bridge client config (bridge.yaml) in HOME (~/.zcn) folder")

	// To avoid noise in output of subsequent operations
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

	cmd := "./zwallet bridge-client-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --consensusthreshold " + fmt.Sprintf("%.4f", consensusthreshold) +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	for _, opt := range opts {
		cmd = fmt.Sprintf(" %s --%s %s ", cmd, opt.name, opt.value)
	}

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// cmd: bridge-owner-init
func bridgeOwnerInit(
	t *test.SystemTest,
	password, ethereumaddress, bridgeaddress, wzcnaddress, authorizersaddress, ethereumnodeurl string,
	gaslimit, value int64,
	opts ...*Option,
) ([]string, error) {
	t.Logf("Init bridge owner config (owner.yaml) in HOME (~/.zcn) folder")

	cmd := "./zwallet bridge-owner-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --authorizersaddress " + authorizersaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	for _, opt := range opts {
		cmd = fmt.Sprintf(" %s --%s %s ", cmd, opt.name, opt.value)
	}

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func createDefaultClientBridgeConfig(t *test.SystemTest) ([]string, error) {
	return bridgeClientInit(t,
		"password",
		"0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947",
		"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
		"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
		"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
		0.75,
		300000,
		0,
	)
}

func runCreateBridgeClientTestConfig(
	t *test.SystemTest,
	password, ethereumaddress, bridgeaddress, wzcnaddress, ethereumnodeurl string,
	consensusthreshold float64,
	gaslimit, value int64,
	opts ...*Option,
) ([]string, error) {
	cmd := "./zwallet bridge-client-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --consensusthreshold " + fmt.Sprintf("%.4f", consensusthreshold) +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	for _, opt := range opts {
		cmd = fmt.Sprintf(" %s --%s %s ", cmd, opt.name, opt.value)
	}

	return cliutils.RunCommandWithoutRetry(cmd)
}

func WithOption(name, value string) *Option {
	return &Option{
		name:  name,
		value: value,
	}
}

type Option struct {
	name  string
	value string
}
