package docker

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/log"
)

func withTestRoot(path ...string) (string, error) {
	rd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fp := strings.Split(rd, string(os.PathSeparator))
	fp = append(fp, path...)
	return strings.Join(fp, string(os.PathSeparator)), nil
}

type (
	errorLine struct {
		Error       string      `json:"error"`
		ErrorDetail errorDetail `json:"errorDetail"`
	}

	errorDetail struct {
		Message string `json:"message"`
	}
)

func checkBuilding(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		log.Logger.Info(lastLine)
	}

	errLine := &errorLine{}
	if err := json.Unmarshal([]byte(lastLine), errLine); err != nil {
		return err
	}

	if lastLine == "" {
		return errors.New("unexpected last line")
	}

	return nil
}
