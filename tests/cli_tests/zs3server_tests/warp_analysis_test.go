package zs3servertests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWarpAnalysis(t *testing.T) {
	// Check if warp binary is available
	if _, err := os.Stat("../warp"); os.IsNotExist(err) {
		t.Skip("warp binary not available at ../warp, skipping test")
	}

	files, err := filepath.Glob("warp*.csv.zst")
	if err != nil {
		t.Fatalf("Error finding files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No warp*.csv.zst files found, skipping analysis test")
	}

	for _, file := range files {
		fmt.Printf("<%s->\n\n", strings.Repeat("-", 50))
		fmt.Printf("Analyzing %s\n\n", file)
		cmd := exec.Command("../warp", "analyze", "--analyze.op=GET", "--analyze.v", file)

		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("warp analyze returned non-zero for %s: %v\nOutput:\n%s", file, err, stdoutStderr)
		}

		fmt.Printf("Command output for %s:\n%s\n", file, stdoutStderr)

		fmt.Printf("<%s->\n\n", strings.Repeat("-", 50))
	}
}
