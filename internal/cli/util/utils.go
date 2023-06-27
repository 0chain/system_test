package cliutils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/0chain/system_test/internal/cli/util/specific"

	"github.com/sirupsen/logrus"
)

var Logger = getLogger()

func RunCommandWithoutRetry(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

	Logger.Debugf("Command [%v] exited with error [%v] and output [%v]", commandString, err, sanitizeOutput(rawOutput))

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

	var count int
	for {
		count++
		output, err := RunCommandWithoutRetry(commandString)

		errorMessage := err.Error()

		t.Log("Error message: ", errorMessage)

		if strings.Contains(errorMessage, "Download failed for block 0") {
			t.Log("Download failed for block 0, not retrying...")
			return output, nil
		}

		if err == nil {
			if count > 1 {
				t.Logf("%sCommand passed on retry [%v/%v]. Output: [%v]\n", green, count, maxAttempts, strings.Join(output, " -<NEWLINE>- "))
			}
			return output, nil
		} else if count < maxAttempts {
			t.Logf("%sCommand failed on attempt [%v/%v] due to error [%v]. Output: [%v]\n", yellow, count, maxAttempts, err, strings.Join(output, " -<NEWLINE>- "))
			time.Sleep(backoff)
		} else {
			t.Logf("%sCommand failed on final attempt [%v/%v] due to error [%v]. Command String: [%v] Output: [%v]\n", red, count, maxAttempts, err, commandString, strings.Join(output, " -<NEWLINE>- "))

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
