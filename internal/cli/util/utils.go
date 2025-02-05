package cliutils

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/0chain/system_test/internal/cli/util/specific"

	"github.com/sirupsen/logrus"
)

var Logger = getLogger()

var (
	WalletMutex sync.Mutex
	Wallets     []json.RawMessage
	WalletIdx   int64
	mu          sync.Mutex
)

func GetWallets() []json.RawMessage {
	return Wallets
}

func SetWallets(wallets []json.RawMessage) {
	WalletMutex.Lock()
	defer WalletMutex.Unlock()
	Wallets = wallets
}

type Configuration struct {
	Server      string
	HostPort    string
	AccessKey   string
	SecretKey   string
	Concurrent  string
	ObjectSize  string
	ObjectCount string
}

type McConfiguration struct {
	Server          string
	HostPort        string
	AccessKey       string
	SecretKey       string
	Concurrent      string
	SecondaryPort   string
	SecondaryServer string
	UseCommand      bool
}

func RunCommandWithoutRetry(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)
	var commandStringForLog string

	// Redact keys before logging
	if strings.Contains(commandString, "--access-key") {
		index := strings.Index(commandString, "--access-key")
		// get next single word after "--access-key"
		accessKey := strings.Fields(commandString[index:])[1]
		commandStringForLog = strings.ReplaceAll(commandString, accessKey, "****")
	}
	if strings.Contains(commandString, "--secret-key") {
		index := strings.Index(commandString, "--secret-key")
		// get next single word after "--secret-key"
		secretKey := strings.Fields(commandString[index:])[1]
		commandStringForLog = strings.ReplaceAll(commandString, secretKey, "****")
	}
	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandStringForLog, err, sanitizeOutput(rawOutput))

	return sanitizeOutput(rawOutput), err
}

func RunCommandWithRawOutput(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandString, err, string(rawOutput))

	output := strings.Split(string(rawOutput), "\n")

	return output, err
}

func RunCommand(t *test.SystemTest, commandString string, maxAttempts int, backoff time.Duration) ([]string, error) {
	red := "\033[31m"
	yellow := "\033[33m"
	green := "\033[32m"
	var commandStringForLog string

	var count int
	for {
		count++
		output, err := RunCommandWithoutRetry(commandString)

		if err == nil {
			if count > 1 {
				t.Logf("%sCommand passed on retry [%v/%v]. Output: [%v]\n", green, count, maxAttempts, strings.Join(output, " -<NEWLINE>- "))
			}
			return output, nil
		} else if count < maxAttempts {
			t.Logf("%sCommand failed on attempt [%v/%v] due to error [%v]. Output: [%v]\n", yellow, count, maxAttempts, err, strings.Join(output, " -<NEWLINE>- "))
			time.Sleep(backoff)
		} else {
			// Redact keys before logging
			if strings.Contains(commandString, "--access-key") {
				index := strings.Index(commandString, "--access-key")
				// get next single word after "--access-key"
				accessKey := strings.Fields(commandString[index:])[1]
				commandStringForLog = strings.ReplaceAll(commandString, accessKey, "****")
			}
			if strings.Contains(commandString, "--secret-key") {
				index := strings.Index(commandString, "--secret-key")
				// get next single word after "--secret-key"
				secretKey := strings.Fields(commandString[index:])[1]
				commandStringForLog = strings.ReplaceAll(commandString, secretKey, "****")
			}
			t.Logf("%sCommand failed on final attempt [%v/%v] due to error [%v]. Command String: [%v] Output: [%v]\n", red, count, maxAttempts, err, commandStringForLog, strings.Join(output, " -<NEWLINE>- "))

			if err != nil {
				t.Logf("%sThe verbose output for the command is:", red)
				commandString = strings.Replace(commandString, "--silent", "", 1)
				out, _ := RunCommandWithoutRetry(commandString) // Only for logging!
				for _, line := range out {
					t.Logf("%s%s", red, line)
				}
			}

			return output, err
		}
	}
}

func StartCommand(t *test.SystemTest, commandString string, maxAttempts int, backoff time.Duration) (cmd *exec.Cmd, err error) {
	var count int
	for {
		count++
		cmd, err := StartCommandWithoutRetry(commandString)

		if err == nil {
			if count > 1 {
				t.Logf("Command started on retry [%v/%v].", count, maxAttempts)
			}
			return cmd, err
		} else if count < maxAttempts {
			t.Logf("Command failed on attempt [%v/%v] due to error [%v]\n", count, maxAttempts, err)
			t.Logf("Sleeping for backoff duration: %v\n", backoff)
			_ = cmd.Process.Kill()
			time.Sleep(backoff)
		} else {
			t.Logf("Command failed on final attempt [%v/%v] due to error [%v].\n", count, maxAttempts, err)
			_ = cmd.Process.Kill()
			return cmd, err
		}
	}
}

func StartCommandWithoutRetry(commandString string) (cmd *exec.Cmd, err error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]
	sanitizedArgs := sanitizeArgs(args)

	cmd = exec.Command(commandName, sanitizedArgs...)
	specific.Setpgid(cmd)
	err = cmd.Start()

	return cmd, err
}

func RandomAlphaNumericString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret)
}

func Wait(t *test.SystemTest, duration time.Duration) {
	t.Logf("Waiting %s...", duration)
	time.Sleep(duration)
}

func sanitizeOutput(rawOutput []byte) []string {
	output := strings.Split(string(rawOutput), "\n")
	var sanitizedOutput []string

	for _, lineOfOutput := range output {
		uniqueOutput := strings.Join(unique(strings.Split(lineOfOutput, "\r")), " ")
		trimmedOutput := strings.TrimSpace(uniqueOutput)
		if trimmedOutput != "" {
			sanitizedOutput = append(sanitizedOutput, trimmedOutput)
		}
	}

	return unique(sanitizedOutput)
}

func unique(slice []string) []string {
	var uniqueOutput []string
	existingOutput := make(map[string]bool)

	for _, element := range slice {
		trimmedElement := strings.TrimSpace(element)
		if _, existing := existingOutput[trimmedElement]; !existing {
			existingOutput[trimmedElement] = true
			uniqueOutput = append(uniqueOutput, trimmedElement)
		}
	}

	return uniqueOutput
}

func executeCommand(commandName string, args []string) ([]byte, error) {
	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	return rawOutput, err
}

func sanitizeArgs(args []string) []string {
	var sanitizedArgs []string
	for _, arg := range args {
		sanitizedArgs = append(sanitizedArgs, strings.ReplaceAll(arg, "\"", ""))
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

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEBUG")), "true") {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

func Contains(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// GetSubPaths will return [/ /a /a/b /a/b/c /a/b/c/d /a/b/c/d/e] for
// path /a/b/c/d/e
func GetSubPaths(p string) (paths []string, err error) {
	if !filepath.IsAbs(p) {
		return nil, fmt.Errorf("path %s is not absolute", p)
	}

	p = filepath.Clean(p)
	splittedPaths := strings.Split(p, "/")
	for i := 0; i < len(splittedPaths); i++ {
		subPath := filepath.Join(string(os.PathSeparator), strings.Join(splittedPaths[0:i+1], string(os.PathSeparator)))
		paths = append(paths, subPath)
	}

	return
}

func ReadFile(testSetup *testing.T) Configuration {
	var config Configuration

	file, err := os.Open("hosts.yaml")
	if err != nil {
		testSetup.Fatalf("Error opening hosts.yaml file: %v\n", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var hosts map[string]interface{}
	err = decoder.Decode(&hosts)
	if err != nil {
		testSetup.Fatalf("Error decoding hosts.yaml file: %v\n", err)
	}

	config.AccessKey = hosts["access_key"].(string)
	config.SecretKey = hosts["secret_key"].(string)
	port := hosts["port"].(int)
	concurrent := hosts["concurrent"].(int)
	config.ObjectSize = hosts["object_size"].(string)
	objectCount := hosts["object_count"].(int)
	config.Server = hosts["server"].(string)
	config.HostPort = strconv.FormatInt(int64(port), 10)
	config.ObjectCount = strconv.FormatInt(int64(objectCount), 10)
	config.Concurrent = strconv.FormatInt(int64(concurrent), 10)
	return config
}

func ReadFileAllocation() (data, parity, lock, accessKey, secretKey string) {
	file, err := os.Open("allocation.yaml")
	if err != nil {
		log.Fatalf("Error opening allocation.yaml file: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var allocationData map[string]interface{}

	if err := decoder.Decode(&allocationData); err != nil {
		log.Printf("Error decoding allocation.yaml file: %v", err)
	}
	data_int := allocationData["data"].(int)
	parity_int := allocationData["parity"].(int)
	lock_int := allocationData["lock"].(int)
	accessKey = allocationData["access_key"].(string)
	secretKey = allocationData["secret_key"].(string)

	data = strconv.FormatInt(int64(data_int), 10)
	parity = strconv.FormatInt(int64(parity_int), 10)
	lock = strconv.FormatInt(int64(lock_int), 10)
	return data, parity, lock, accessKey, secretKey
}

func AppendToFile(filename, data string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(data); err != nil {
		return err
	}
	return nil
}

func KillProcess() (int, error) {
	// Create a command to get the PID of the process listening on the specified port
	cmd := exec.Command("lsof", "-t", "-i", ":9000")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error running lsof -i command: %v", err)
	}
	pidStr := strings.TrimSpace(string(out))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found for port %s", "9000")
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("error converting PID to integer: %v", err)
	}

	// Create a command to kill the process identified by PID
	killCmd := exec.Command("kill", strconv.Itoa(pid)) // #nosec G204: subprocess launched with tainted input

	if err := killCmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to kill process with PID %d: %w ", pid, err)
	}

	return pid, nil
}

func SplitCmdString(cmdString string) ([]string, error) {
	return []string{"sh", "-c", cmdString}, nil
}

func LogOutput(stdout io.Reader, t *test.SystemTest) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		t.Logf("[MinIO stdout] %s", scanner.Text())
	}
}

func GetAllocationID(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening allocation.txt file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	allocationID := scanner.Text()
	return allocationID, nil
}

func ReadFileMC(testSetup *testing.T) McConfiguration {
	var config McConfiguration

	file, err := os.Open("mc_hosts.yaml")
	if err != nil {
		testSetup.Fatalf("Error opening hosts.yaml file: %v\n", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var hosts map[string]interface{}
	err = decoder.Decode(&hosts)
	if err != nil {
		testSetup.Fatalf("Error decoding mc_hosts.yaml file: %v\n", err)
	}

	config.AccessKey = hosts["access_key"].(string)
	config.SecretKey = hosts["secret_key"].(string)
	port := hosts["port"].(int)
	concurrent := hosts["concurrent"].(int)
	config.Server = hosts["server"].(string)
	config.SecondaryServer = hosts["secondary_server"].(string)
	s_port := hosts["secondary_port"].(int)
	use_command, ok := hosts["use_command"].(bool)

	if !ok {
		use_command = false
	}
	config.UseCommand = use_command
	config.HostPort = strconv.FormatInt(int64(port), 10)
	config.SecondaryPort = strconv.FormatInt(int64(s_port), 10)
	config.Concurrent = strconv.FormatInt(int64(concurrent), 10)
	return config
}

func MigrateFromS3migration(t *test.SystemTest, params string) ([]string, error) {
	commandGenerated := fmt.Sprintf("../s3migration migrate  %s", params)
	t.Log(commandGenerated)
	return RunCommand(t, commandGenerated, 1, time.Hour*2)
}

func EscapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func CreateParams(params map[string]interface{}) string {
	var builder strings.Builder

	for k, v := range params {
		if v == nil {
			_, _ = builder.WriteString(fmt.Sprintf("--%s ", k))
		} else if reflect.TypeOf(v).String() == "bool" {
			_, _ = builder.WriteString(fmt.Sprintf("--%s=%v ", k, v))
		} else {
			_, _ = builder.WriteString(fmt.Sprintf("--%s %v ", k, v))
		}
	}
	return strings.TrimSpace(builder.String())
}

func ExecuteFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return ExecuteFaucetWithTokensForWallet(t, EscapedTestName(t), cliConfigFilename, tokens)
}

// ExecuteFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func ExecuteFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	t.Logf("Executing faucet...")
	return RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
		"pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		wallet,
		cliConfigFilename,
	), 3, time.Second*5)
}

func CreateWalletForName(rootPath, name string) {
	walletPath := fmt.Sprintf("%s/config/%s_wallet.json", rootPath, name)

	// check if wallet already exists
	if _, err := os.Stat(walletPath); err == nil {
		return
	}
	WalletMutex.Lock()

	wallet := Wallets[WalletIdx]

	WalletIdx++
	WalletMutex.Unlock()

	err := os.WriteFile(walletPath, wallet, 0600)
	if err != nil {
		fmt.Printf("Error writing file %s: %v\n", walletPath, err)
	} else {
		fmt.Printf("File %s written successfully.\n", walletPath)
	}
}

func SetupAllocation(t *test.SystemTest, cliConfigFilename, rootPath string, extraParams ...map[string]interface{}) string {
	return setupAllocationWithWallet(t, EscapedTestName(t), cliConfigFilename, rootPath, extraParams...)
}

func CreateNewAllocationForWallet(t *test.SystemTest, wallet, cliConfigFilename, rootPath, params string) ([]string, error) {
	t.Log(cliConfigFilename, "configdir path")
	return RunCommand(t, fmt.Sprintf(
		"%s/zbox newallocation %s --silent --wallet %s --configDir %s --config %s --allocationFileName %s",
		rootPath,
		params,
		wallet+"_wallet.json",
		cliConfigFilename,
		"config.yaml",
		wallet+"_allocation.txt",
	), 3, time.Second*5)
}

func setupAllocationWithWallet(t *test.SystemTest, walletName, cliConfigFilename, rootPath string, extraParams ...map[string]interface{}) string {
	// Then create new allocation
	options := map[string]interface{}{"size": "10000000", "lock": "5"}

	// Add additional parameters if available
	// Overwrite with new parameters when available
	for _, params := range extraParams {
		for k, v := range params {
			options[k] = v
		}
	}
	// First create a wallet and run faucet command
	CreateWalletForName(rootPath, walletName)

	output, err := CreateNewAllocationForWallet(t, walletName, cliConfigFilename, rootPath, CreateParams(options))
	defer func() {
		fmt.Printf("err: %v\n", err)
	}()
	require.NoError(t, err, "create new allocation failed", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	// Get the allocation ID and return it
	allocationID, err := getAllocationID(output[0])
	require.Nil(t, err, "could not get allocation ID", strings.Join(output, "\n"))

	return allocationID
}

var (
	createAllocationRegex = regexp.MustCompile(`^Allocation created: (.+)$`)
)

func getAllocationID(str string) (string, error) {
	match := createAllocationRegex.FindStringSubmatch(str)
	if len(match) < 2 {
		return "", errors.New("allocation match not found")
	}
	return match[1], nil
}

func GetmigratedDataID(output []string) (totalMigrated int, totalCount int, err error) {

	pattern := `total count :: (\d+)`
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(strings.Join(output, "\n"))

	pattern2 := `Total migrated objects :: (\d+)`
	re2 := regexp.MustCompile(pattern2)
	match_2 := re2.FindStringSubmatch(strings.Join(output, "\n"))

	if len(match) > 1 && len(match_2) > 1 {
		totalCount, err = strconv.Atoi(match[1])
		if err != nil {
			return
		}

		totalMigrated, err = strconv.Atoi(match_2[1])
		if err != nil {
			return
		}

		return totalMigrated, totalCount, nil
	}

	return 0, 0, errors.New("no match found")
}
