package cli_utils

import (
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

func RunCommandWithRetry(commandString string, maxAttempts int) ([]string, error) {
	var count = 0
	for {
		count++
		output, err := RunCommand(commandString)
		if err == nil || count > maxAttempts {
			return output, err
		}
		Logger.Infof("Upload failed on attempt [%v/%v] due to error [%v] and output [%v]", count, maxAttempts, err, strings.Join(output, "\n"))
		time.Sleep(time.Second * 5)
	}
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
	output := strings.FieldsFunc(string(rawOutput), newlineSplit)
	var sanitizedOutput []string
	existingOutput := make(map[string]bool)

	for _, lineOfOutput := range output {
		var trimmedOutput = strings.TrimSpace(lineOfOutput)
		if trimmedOutput != "" {
			if _, existing := existingOutput[trimmedOutput]; !existing {
				existingOutput[trimmedOutput] = true
				sanitizedOutput = append(sanitizedOutput, trimmedOutput)
			}
		}
	}

	return sanitizedOutput
}

func newlineSplit(r rune) bool {
	return r == '\n' || r == '\r'
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
