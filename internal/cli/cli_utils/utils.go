package cli_utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/cli/cli_model"
	"github.com/stretchr/testify/assert"
)

func RunCommand(command string) ([]string, error) {
	r := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	fullCommand := r.FindAllString(command, -1)
	commandName := fullCommand[0]
	args := fullCommand[1:]

	for index, arg := range args {
		args[index] = strings.Replace(arg, "\"", "", -1)
	}
	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	output := strings.Split(strings.TrimSpace(string(rawOutput)), "\n")

	return output, err
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

func RegisterWallet(cliConfigFilename string) ([]string, error) {
	return RunCommand("./zbox register --silent --configDir ./config --config " + cliConfigFilename)
}

func GetBalance(cliConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet getbalance --silent --configDir ./config --config " + cliConfigFilename)
}

func GetWallet(t *testing.T, cliConfigFilename string) (*cli_model.Wallet, error) {
	output, err := RunCommand("./zbox getwallet --json --silent --configDir ./config --config " + cliConfigFilename)

	if err != nil {
		return nil, err
	}

	assert.Equal(t, 1, len(output))

	var wallet *cli_model.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func ExecuteFaucet(cliConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet faucet --methodName pour --tokens 1 --input {} --silent --configDir ./config --config " + cliConfigFilename)
}

func VerifyTransaction(cliConfigFilename string, txn string) ([]string, error) {
	return RunCommand("./zwallet verify --silent --hash " + txn + " --configDir ./config --config " + cliConfigFilename)
}

func NewAllocation(cliConfigFilename string, lock float64, options map[string]string) ([]string, error) {
	cmd := fmt.Sprintf("./zbox newallocation --lock %v --silent --configDir ./config --config %s ", lock, cliConfigFilename)
	for key, option := range options {
		cmd += fmt.Sprintf(" --%s %s ", key, option)
	}
	return RunCommand(cmd)
}

func WritePoolInfo() ([]string, error) {
	return RunCommand("./zbox wp-info --silent")
}

func ChallengePoolInfo(allocationID string) ([]string, error) {
	return RunCommand("./zbox cp-info --allocation " + allocationID + " --silent")
}
