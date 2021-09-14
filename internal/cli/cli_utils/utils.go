package cli_utils

import (
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func RunCommand(commandString string) ([]string, error) {
	command := parseCommand(commandString)
	commandName := command[0]
	args := command[1:]

	sanitizedArgs := sanitizeArgs(args)
	rawOutput, err := executeCommand(commandName, sanitizedArgs)

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
	output := strings.Split(strings.TrimSpace(string(rawOutput)), "\n")
	var sanitizedOutput []string

	for _, str := range output {
		if strings.Trim(str, " ") != "" {
			sanitizedOutput = append(sanitizedOutput, strings.Trim(str, " "))
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
