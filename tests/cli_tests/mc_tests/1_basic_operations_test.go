package cli_tests

import (
	"net"
	"os"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
)

func TestZs3Server(testSetup *testing.T) {
	// Check if mc binary is available, skip if not
	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		testSetup.Skip("mc binary not available at ../mc, skipping test")
	}

	// Check if ZS3 server is reachable at the default mc "play" alias endpoint
	// The mc tests use local mc aliases that point to a ZS3/MinIO server
	// Try connecting to localhost:9100 (ZS3 server port, from mc_hosts.yaml)
	conn, err := net.DialTimeout("tcp", "localhost:9100", 5*time.Second)
	if err != nil {
		testSetup.Skipf("ZS3/MinIO server not available at localhost:9100, skipping test: %v", err)
	}
	conn.Close()

	t := test.NewSystemTest(testSetup)

	defer func() {
		_, err := cli_utils.RunCommand(t, "rm -rf a.txt", 1, time.Hour*2)
		if err != nil {
			t.Logf("Error while deferring command: %v", err)
		}
	}()

	// listing the buckets in the command
	t.RunSequentially("Should list the buckets", func(t *test.SystemTest) {
		output, _ := cli_utils.RunCommand(t, "../mc ls play", 1, time.Hour*2)
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
