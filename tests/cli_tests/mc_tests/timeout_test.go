package cli_tests

import (
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	timeout := time.Duration(200 * time.Minute)
	os.Setenv("GO_TEST_TIMEOUT", timeout.String())
	code := m.Run()
	os.Exit(code)
}
