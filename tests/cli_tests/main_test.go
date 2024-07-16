package cli_tests

import (
	"os"
	"testing"
)


func TestMain(m *testing.M) {
	exitRun := m.Run()
	os.Exit(exitRun)
}
