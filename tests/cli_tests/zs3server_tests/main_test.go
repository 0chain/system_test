package zs3servertests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	exitRun := m.Run()
	os.Exit(exitRun)
}