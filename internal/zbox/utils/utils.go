package zbox_utils

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

func RunCommand(command string) ([]string, error) {
	fullCommand := strings.Split(strings.TrimSpace(command), " ")
	commandName := fullCommand[0]
	args := fullCommand[1:]

	cmd := exec.Command(commandName, args...)
	rawOutput, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("failed to execute the command: %v", err)
		return nil, errors.Wrap(err, "failed to execute command")
	}

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
