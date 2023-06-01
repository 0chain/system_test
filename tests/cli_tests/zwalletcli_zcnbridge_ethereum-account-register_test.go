package cli_tests

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"gopkg.in/errgo.v2/errors"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

const (
	address  = "C49926C4124cEe1cbA0Ea94Ea31a6c12318df947"
	mnemonic = "tag volcano eight thank tide danger coast health above argue embrace heavy"
	password = "password"
)

func TestEthRegisterAccount(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Register ethereum account in local key storage")

	t.RunSequentially("Register ethereum account in local key storage", func(t *test.SystemTest) {
		deleteDefaultAccountInStorage(t, address)
		output, err := importAccount(t, password, mnemonic, false)
		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)
	})

	t.RunSequentially("List ethereum account registered in local key storage", func(t *test.SystemTest) {
		deleteDefaultAccountInStorage(t, address)
		output, err := importAccount(t, password, mnemonic, false)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)

		output, err = listAccounts(t, false)
		require.Nil(t, err, "error trying to register ethereum account", strings.Join(output, "\n"))
		require.Greater(t, len(output), 0)
		require.Contains(t, output[len(output)-1], address)

		deleteDefaultAccountInStorage(t, address)
	})
}

func importAccount(t *test.SystemTest, password, mnemonic string, retry bool) ([]string, error) {
	t.Logf("Register ethereum account using mnemonic and protected with password...")
	output, err := createWallet(t, configPath)
	require.Nil(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))
	cmd := fmt.Sprintf(
		"./zwallet bridge-import-account --password %s --mnemonic \"%s\" --silent "+
			"--configDir ./config --config %s --wallet %s --path %s",
		password,
		mnemonic,
		configPath,
		escapedTestName(t)+"_wallet.json",
		configDir,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAccounts(t *test.SystemTest, retry bool) ([]string, error) {
	t.Logf("List ethereum accounts...")
	cmd := fmt.Sprintf("./zwallet bridge-list-accounts --path %s", configDir)
	cmd += fmt.Sprintf(" --wallet %s --configDir ./config --config %s ", escapedTestName(t)+"_wallet.json", configPath)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func deleteDefaultAccountInStorage(t *test.SystemTest, address string) {
	keyDir := path.Join(configDir, "wallets")
	if _, err := os.Stat(keyDir); err != nil {
		t.Skipf("wallets folder at location is missing: %s", keyDir)
	}

	err := filepath.Walk(keyDir, func(path string, info fs.FileInfo, err error) error {
		if e := IsNil(info); e != nil {
			t.Logf("path %s is nil", path)
			return nil
		}

		if !info.IsDir() {
			require.NoError(t, err)
			if strings.Contains(strings.ToLower(path), strings.ToLower(address)) {
				err = os.Remove(path)
				require.NoError(t, err)
			}
		}
		return nil
	})

	require.NoError(t, err)
}

func getConfigDir() string {
	var configDir string
	curr, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	configDir = filepath.Join(curr, "config")
	return configDir
}

func IsNil(value interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return errors.New("result must be a pointer")
	}

	val = val.Elem()
	if !val.CanAddr() {
		return errors.New("result must be addressable (a pointer)")
	}

	return nil
}
