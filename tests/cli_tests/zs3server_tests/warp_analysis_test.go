package zs3servertests

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWarpAnalysis(t *testing.T) {
	files, err := filepath.Glob("warp*.csv.zst")
	if err != nil {
		t.Fatalf("Error finding files: %v", err)
	}

	for _, file := range files {
		fmt.Printf("<%s->\n\n", strings.Repeat("-", 50))
		fmt.Printf("Analyzing %s\n\n", file)
		cmd := exec.Command("../warp", "analyze", "--analyze.op=GET", "--analyze.v", file)

		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Error executing command for %s: %v\nOutput:\n%s", file, err, stdoutStderr)
		}

		fmt.Printf("Command output for %s:\n%s\n", file, stdoutStderr)

		fmt.Printf("<%s->\n\n", strings.Repeat("-", 50))
	}
}
