package cli_utils

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var Logger = getLogger()

func RunCommand(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandString, err, string(rawOutput))

	return sanitizeOutput(rawOutput), err
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

func sanitizeOutput(rawOutput []byte) []string {
	output := strings.Split(string(rawOutput), "\n")
	var sanitizedOutput []string

	for _, lineOfOutput := range output {
		if strings.TrimSpace(lineOfOutput) != "" {
			sanitizedOutput = append(sanitizedOutput, strings.TrimSpace(lineOfOutput))
		}
	}

	return sanitizedOutput
}

func executeCommand(commandName string, args []string) ([]byte, error) {
	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	return rawOutput, err
}

func sanitizeArgs(args []string) []string {
	var sanitizedArgs []string
	for _, arg := range args {
		sanitizedArgs = append(sanitizedArgs, strings.Replace(arg, "\"", "", -1))
	}

	return sanitizedArgs
}

func parseCommand(command string) []string {
	commandArgSplitter := regexp.MustCompile(`[^\s"]+|"([^"]*)"`)
	fullCommand := commandArgSplitter.FindAllString(command, -1)

	return fullCommand
}

func getLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = os.Stdout

	logger.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	if strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG"))) == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

func RegisterWallet(cliConfigFilename string) ([]string, error) {
	return RunCommand("./zbox register --silent --configDir ./config --config " + cliConfigFilename)
}

func GetBalance(cliConfigFilename string) ([]string, error) {
	return RunCommand("./zwallet getbalance --silent --configDir ./config --config " + cliConfigFilename)
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
