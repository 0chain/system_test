package utils

import (
	"encoding/json"
	"github.com/0chain/system_test/internal/model"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func RunCommand(command string) ([]string, error) {
	fullCommand := strings.Split(strings.TrimSpace(command), " ")
	commandName := fullCommand[0]
	args := fullCommand[1:]

	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	output := strings.Split(strings.TrimSpace(string(rawOutput)), "\n")

	return output, err
}

func RegisterWallet(walletConfigFilename string) ([]string, error) {
	return RunCommand("./zbox register --silent --wallet " + walletConfigFilename)
}

func GetBalance(walletConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet getbalance --silent --wallet " + walletConfigFilename)
}

func GetWallet(t *testing.T, wallet model.Wallet, walletConfigFilename string) error {
	output, err := RunCommand("./zbox getwallet --json --silent --wallet " + walletConfigFilename)

	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(output))

	return json.Unmarshal([]byte(output[0]), &wallet)
}

func ExecuteFaucet(walletConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent --wallet " + walletConfigFilename)
}

func VerifyTransaction(walletConfigFilename string, txn string) ([]string, error) {
	return RunCommand("./zwallet verify --silent --wallet " + walletConfigFilename + " --hash " + txn)
}

func RandomAlphaNumericString(n int) string {
	rand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
