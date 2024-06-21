// create a tear down for all tests
package zs3servertests

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
)

var allocationId string

func TestMain(m *testing.M) {
	globalSetup()
	timeout := time.Duration(60 * time.Minute)
	os.Setenv("GO_TEST_TIMEOUT", timeout.String())
	code := m.Run()
	globalTearDown()
	os.Exit(code)
}

func hasParentDir(path string) bool {
	return path != filepath.Dir(path)
}

func globalSetup() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current working directory:", err)
	}

	// Check if a parent directory exists
	if !hasParentDir(currentDir) {
		log.Fatal("Script must be run from a directory with a parent")
	}

	requiredCommands := map[string]string{
		"mc":    "../mc",
		"zbox":  "../zbox",
		"minio": "../minio",
		"warp":  "../warp",
	}

	for cmd, path := range requiredCommands {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatal(cmd + " is not installed")
			os.Exit(1)
		} else {
			log.Fatal(cmd + " is  installed")
		}

		if requiredCommands[cmd] == requiredCommands["warp"] {
			log.Print("All required commands are installed")
		} else {
			log.Print("Checking for next command")

		}
	}

	// create allocation from allocation.yaml file
	data, parity, lock := cliutils.Read_file_allocation()
	cmd := exec.Command("../zbox", "newallocation", "--lock", lock, "--data", data, "--parity", parity)

	// get the allocation id for created

	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatal("Error creating allocation: ", err)
		os.Exit(1)
	} else {
		log.Fatal("Allocation created successfully")
	}

	// use regex
	re := regexp.MustCompile(`Allocation created:\s*([a-f0-9]+)`)

	match := re.FindStringSubmatch(string(output))
	if len(match) > 1 {
		allocationId = match[1]
	}

	os.Setenv("MINIO_ROOT_USER", "someminiouser")
	os.Setenv("MINIO_ROOT_PASSWORD", "someminiopassword")

	cmd = exec.Command("./minio", "gateway", "zcn", "--console-address", ":8000")

	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	log.Print("Minio server started")

	println("Global setup code executed")

}

func globalTearDown() {
	println("Global teardown code Executing .......")
	err := exec.Command("../zbox", "delete", "--allocation", allocationId, "--remotepath", "/").Run()
	if err != nil {
		log.Fatal("Error deleting allocation: ", err)
		os.Exit(1)
	} else {
		log.Print("Allocation deleted successfully")
	}

}
