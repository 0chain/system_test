package cli_tests

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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

func PrepareBridgeClient() error {
	_, err := prepareBridgeClientConfig()
	if err != nil {
		return err
	}

	_, err = prepareBridgeClientWallet()
	if err != nil {
		return err
	}

	return nil
}

// Tests prerequisites
func prepareBridgeClientConfig() ([]string, error) {
	return runCreateBridgeClientTestConfig(
		"password",
		"0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947",
		"0xF26B52df8c6D9b9C20bfD7819Bed75a75258c7dB",
		"0x930E1BE76461587969Cb7eB9BFe61166b1E70244",
		"https://ropsten.infura.io/v3/22cb2849f5f74b8599f3dc2a23085bd4",
		75,
		300000,
		0,
		WithOption(OptionConfigFolder, configDir),
		WithOption(OptionBridgeConfigFile, bridgeClientConfigFile),
	)
}

// Use it to import account to the given home folder
func prepareBridgeClientWallet() ([]string, error) {
	cmd := fmt.Sprintf(
		"./zwallet bridge-import-account --%s %s --%s \"%s\" --%s %s",
		OptionConfigFolder, configDir,
		OptionMnemonic, mnemonic,
		OptionKeyPassword, password,
	)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// cmd: bridge-client-init
func TestBridgeClientInit(t *testing.T) {
	t.Run("Init bridge client config to default path and file", func(t *testing.T) {
		output, err := createDefaultClientBridgeConfig(t)

		customPath := filepath.Join(getConfigDir(), DefaultConfigBridgeFileName)

		require.Nil(t, err, "error trying to create an initial client bridge config", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", customPath), output[len(output)-1])
	})

	customPath := path.Join(getConfigDir(), "test")

	t.Run("Init bridge client config to custom path and default file", func(t *testing.T) {
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
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", customPath), output[len(output)-1])
	})

	t.Run("Init bridge client config to custom path and custom config file", func(t *testing.T) {
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
		require.Equal(t, fmt.Sprintf("Client client config file was saved to %s", customPath), output[len(output)-1])
	})
}

// cmd: bridge-owner-init
func TestBridgeOwnerInit(t *testing.T) {
	t.Run("Init bridge owner config to default path and file", func(t *testing.T) {
		output, err := bridgeOwnerInit(
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

		customPath := filepath.Join(getConfigDir(), DefaultConfigOwnerFileName)

		require.Nil(t, err, "error trying to create an initial owner config", strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("Owner config file was saved to %s", customPath), output[len(output)-1])
	})
}

// cmd: bridge-client-init
func bridgeClientInit(
	t *testing.T,
	password, ethereumaddress, bridgeaddress, wzcnaddress, ethereumnodeurl string,
	consensusthreshold float64,
	gaslimit, value int64,
	opts ...*Option,
) ([]string, error) {
	t.Logf("Init bridge client config (bridge.yaml) in HOME (~/.zcn) folder")

	cmd := "./zwallet bridge-client-init" +
		" --password " + password +
		" --ethereumaddress " + ethereumaddress +
		" --bridgeaddress " + bridgeaddress +
		" --wzcnaddress " + wzcnaddress +
		" --ethereumnodeurl " + ethereumnodeurl +
		" --consensusthreshold " + fmt.Sprintf("%.4f", consensusthreshold) +
		" --gaslimit " + strconv.FormatInt(gaslimit, 10) +
		" --value " + strconv.FormatInt(value, 10)

	for _, opt := range opts {
		cmd = fmt.Sprintf(" %s --%s %s ", cmd, opt.name, opt.value)
	}

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}

// cmd: bridge-owner-init
func bridgeOwnerInit(
	t *testing.T,
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

	for _, opt := range opts {
		cmd = fmt.Sprintf(" %s --%s %s ", cmd, opt.name, opt.value)
	}

	t.Log(cmd)

	return cliutils.RunCommandWithoutRetry(cmd)
}

func createDefaultClientBridgeConfig(t *testing.T) ([]string, error) {
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
