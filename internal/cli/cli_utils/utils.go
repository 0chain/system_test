package cli_utils

import (
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"time"
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
