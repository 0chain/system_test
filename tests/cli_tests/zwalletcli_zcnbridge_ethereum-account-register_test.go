package cli_tests

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/errgo.v2/errors"

	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

const (
	address       = "C49926C4124cEe1cbA0Ea94Ea31a6c12318df947"
	mnemonic      = "tag volcano eight thank tide danger coast health above argue embrace heavy"
	password      = "password"
	walletsFolder = "wallets"
)

func TestEthRegisterAccount(t *testing.T) {
	t.Parallel()

	zwallet := func(cmd, mnemonic, password string, retry bool) ([]string, error) {
		t.Logf("Register ethereum account using mnemonic and protected with password in HOME (~/.zcn) folder")

		run := fmt.Sprintf(
			"./zwallet %s --password %s --mnemonic \"%s\" --path %s --wallet %s --silent",
			cmd,
			password,
			mnemonic,
			configDir,
			escapedTestName(t)+"_wallet.json",
		)

		if retry {
			return cliutils.RunCommand(t, run, 3, time.Second*2)
		} else {
			return cliutils.RunCommandWithoutRetry(run)
		}
	}

	zwalletList := func(cmd string, retry bool) ([]string, error) {
		t.Logf("List ethereum account registered in local key chain in HOME (~/.zcn) folder")

		run := fmt.Sprintf("./zwallet %s --path %s --silent", cmd, configDir)

		if retry {
			return cliutils.RunCommand(t, run, 3, time.Second*2)
		} else {
			return cliutils.RunCommandWithoutRetry(run)
		}
	}

	t.Run("Register ethereum account in local key storage", func(t *testing.T) {
		deleteDefaultAccountInStorage(t, address)
		output, err := importAccount(t, zwallet)
		require.Nilf(t, err, "error trying to register ethereum account: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)
	})

	t.Run("List ethereum account registered in local key storage", func(t *testing.T) {
		deleteDefaultAccountInStorage(t, address)
		output, err := importAccount(t, zwallet)
		require.NoError(t, err, "prerequisites failed")
		require.Contains(t, output[len(output)-1], "Imported account 0x"+address)

		output, err = zwalletList("bridge-list-accounts", false)
		require.Nilf(t, err, "error trying to list ethereum accounts in local key storage: %s", strings.Join(output, "\n"))
		require.Contains(t, output[len(output)-1], address)
		deleteDefaultAccountInStorage(t, address)
	})
}

func importAccount(t *testing.T, zwallet func(
	cmd string,
	mnemonic string,
	password string,
	retry bool,
) ([]string, error),
) ([]string, error) {
	t.Logf("Register ethereum account using mnemonic and protected with password...")
	output, err := zwallet(
		"bridge-import-account",
		mnemonic,
		password,
		false,
	)

	return output, err
}

func deleteDefaultAccountInStorage(t *testing.T, address string) {
	keyDir := path.Join(configDir, walletsFolder)
	if _, err := os.Stat(keyDir); err != nil {
		t.Logf("%s folder is missing: %s", walletsFolder, keyDir)
		t.Logf("Creating %s folder", keyDir)
		err := os.MkdirAll(keyDir, os.ModePerm)
		if err != nil {
			t.Skipf("Skipping - could not create %s folder", keyDir)
		}
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
		panic(err)
	}
	configDir = filepath.Join(curr, "config")

	return configDir
}

func getZCNDir() string {
	var configDir string
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	configDir = home + "/.zcn"

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
