package utils

import (
	"encoding/json"
	"fmt"
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

func RegisterWallet(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return RunCommand("./zbox register --silent --wallet " + walletConfigFilename + " --configDir ./temp --config " + cliConfigFilename)
}

func GetBalance(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet getbalance --silent --wallet " + walletConfigFilename + " --configDir ./temp --config " + cliConfigFilename)
}

func GetWallet(t *testing.T, walletConfigFilename string, cliConfigFilename string) (*model.Wallet, error) {
	output, err := RunCommand("./zbox getwallet --json --silent --wallet " + walletConfigFilename + " --configDir ./temp --config " + cliConfigFilename)

	if err != nil {
		return nil, err
	}

	assert.Equal(t, 1, len(output))

	var wallet model.Wallet

	return &wallet, json.Unmarshal([]byte(output[0]), &wallet)
}

func ExecuteFaucet(walletConfigFilename string, cliConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent --wallet " + walletConfigFilename + " --configDir ./temp --config " + cliConfigFilename)
}

func VerifyTransaction(walletConfigFilename string, cliConfigFilename string, txn string) ([]string, error) {
	return RunCommand("./zwallet verify --silent --wallet " + walletConfigFilename + " --hash " + txn + " --configDir ./temp --config " + cliConfigFilename)
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

func CreateMultiSigWallet(walletConfigFilename string, cliConfigFilename string, numSigners, threshold int) ([]string, error) {
	return RunCommand(fmt.Sprintf(
		"./zwallet createmswallet --numsigners %d --threshold %d --silent --wallet %s --configDir ./temp --config %s",
		numSigners, threshold,
		walletConfigFilename,
		cliConfigFilename))
}
