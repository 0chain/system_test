package utils

import (
	"os/exec"
	"strings"
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
