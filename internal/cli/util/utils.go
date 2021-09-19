package cli_utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/0chain/gosdk/core/conf"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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

func GetNetworkConfiguration(configPath string) (map[string]interface{}, error) {
	config, err := getConfig(configPath)
	if err != nil {
		Logger.Error(err)
		return nil, err
	}

	u, err := url.Parse(config.BlockWorker)
	if err != nil {
		Logger.Errorf("failed to parse config.BlockWorker (%s): %v", config.BlockWorker, err)
		return nil, err
	}
	u.Path = path.Join(u.Path, "network")
	res := &map[string]interface{}{}
	err = getJson(u.String(), res)
	if err != nil {
		Logger.Errorf("failed to get configuration from the dns network (%s): %v", u, err)
		return nil, err
	}

	return *res, nil
}

func GetChainConfiguration(configPath string) (map[string]interface{}, error) {
	networkConfig, err := GetNetworkConfiguration(configPath)
	if err != nil {
		return nil, err
	}
	var miners []interface{} = (networkConfig["miners"]).([]interface{})
	if len(miners) == 0 {
		errMsg := fmt.Sprintf("Cannot read miners from the dns network configuration: %v", networkConfig["miners"])
		Logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	u, err := url.Parse(miners[0].(string))
	if err != nil {
		Logger.Errorf("failed to parse miner 0 address (%s): %v", miners[0], err)
		return nil, err
	}
	u.Path = path.Join(u.Path, "v1/config/get")

	res := &map[string]interface{}{}
	err = getYaml(u.String(), res)
	if err != nil {
		Logger.Fatalf("failed to get chain configuration from miner (%s): %v", u.String(), err)
	}

	return *res, nil
}

func getConfig(configPath string) (conf.Config, error) {
	config, err := conf.LoadConfigFile("./config/zbox_config.yaml")
	if err != nil {
		Logger.Fatalf("failed to fetch configuration from the ConfigPath: %v", err)
	}
	return config, err
}

func getJson(url string, target interface{}) error {
	myClient := &http.Client{Timeout: 30 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func getYaml(url string, target interface{}) error {
	myClient := &http.Client{Timeout: 30 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		Logger.Fatal(err)
	}
	return yaml.Unmarshal(b, target)
}
