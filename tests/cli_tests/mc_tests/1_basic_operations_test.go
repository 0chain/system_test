package cli_tests

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
)

func TestZs3Server(testSetup *testing.T) {
	// Require mc binary
	_, err := os.Stat("../mc")
	if os.IsNotExist(err) {
		testSetup.Fatalf("mc binary not available at ../mc")
	}

	// Require mc_hosts.yaml config
	_, err = os.Stat("mc_hosts.yaml")
	if os.IsNotExist(err) {
		testSetup.Fatalf("mc_hosts.yaml config not found")
	}

	config := cli_utils.ReadFileMC(testSetup)

	// Wait for ZS3 server to become reachable (up to 60s)
	var conn net.Conn
	for i := 0; i < 12; i++ {
		conn, err = net.DialTimeout("tcp", config.Server+":"+config.HostPort, 5*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		testSetup.Logf("ZS3 server not available at %s:%s (attempt %d/12), retrying in 5s...", config.Server, config.HostPort, i+1)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		testSetup.Fatalf("ZS3/MinIO server not available at %s:%s after 60s: %v", config.Server, config.HostPort, err)
	}

	t := test.NewSystemTest(testSetup)

	// Set up the zs3 alias pointing to the local ZS3 server
	aliasCmd := "../mc alias set zs3 http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
	aliasOutput, aliasErr := cli_utils.RunCommand(t, aliasCmd, 1, time.Minute)
	if aliasErr != nil {
		t.Fatalf("Failed to set zs3 mc alias: %v\nOutput: %s", aliasErr, strings.Join(aliasOutput, "\n"))
	}

	defer func() {
		_, err := cli_utils.RunCommand(t, "rm -rf a.txt", 1, time.Hour*2)
		if err != nil {
			t.Logf("Error while deferring command: %v", err)
		}
	}()

	// listing the buckets in the command
	t.RunSequentially("Should list the buckets", func(t *test.SystemTest) {
		output, _ := cli_utils.RunCommand(t, "../mc ls zs3", 1, time.Hour*2)
		assert.NotContains(t, output, "error")
	})

	t.RunSequentially("Test Bucket Creation", func(t *test.SystemTest) {
		output, _ := cli_utils.RunCommand(t, "../mc mb custombucket", 1, time.Hour*2)
		assert.Contains(t, output, "Bucket created successfully `custombucket`.")
	})

	t.RunSequentially("Test Copying File Upload", func(t *test.SystemTest) {
		// create a file with content
		_, _ = cli_utils.RunCommand(t, "../mc mb custombucket", 1, time.Hour*2)

		file, err := os.Create("a.txt")
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		defer file.Close()

		_, err = file.WriteString("test")
		if err != nil {
			t.Fatalf("Error writing to file: %v", err)
		}

		output, _ := cli_utils.RunCommand(t, "../mc cp a.txt custombucket", 1, time.Hour*2)

		assert.NotContains(t, output, "../mc: <ERROR>")

		os.Remove("a.txt")
	})

	t.RunSequentially("Test for moving file", func(t *test.SystemTest) {
		_, _ = cli_utils.RunCommand(t, "../mc mb custombucket", 1, time.Hour*2)

		file, err := os.Create("a.txt")
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		defer file.Close()

		_, err = file.WriteString("test")
		if err != nil {
			t.Fatalf("Error writing to file: %v", err)
		}

		_, _ = cli_utils.RunCommand(t, "../mc cp a.txt custombucket", 1, time.Hour*2)

		output, _ := cli_utils.RunCommand(t, "../mc mv custombucket/a.txt custombucket/b", 1, time.Hour*2)
		assert.NotContains(t, output, "../mc: <ERROR>")
	})

	t.RunSequentially("Test for copying file ", func(t *test.SystemTest) {
		output, _ := cli_utils.RunCommand(t, "../mc cp a.txt custombucket", 1, time.Hour*2)

		assert.NotContains(t, output, "../mc: <ERROR>")
	})

	t.RunSequentially("Test for removing file", func(t *test.SystemTest) {
		output, _ := cli_utils.RunCommand(t, "../mc rm custombucket/a.txt", 1, time.Hour*2)
		assert.Contains(t, output, "Removed `custombucket/a.txt`.")
	})
}
