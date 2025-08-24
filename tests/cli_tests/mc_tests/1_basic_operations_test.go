package cli_tests

import (
	"os"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
)

func TestZs3Server(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	if _, err := os.Stat("../mc"); os.IsNotExist(err) {
		t.Fatalf("../mc is not installed")
	} else {
		t.Logf("../mc is installed")
	}

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
		defer file.Close() //nolint:errcheck

		_, err = file.WriteString("test")
		if err != nil {
			t.Fatalf("Error writing to file: %v", err)
		}

		output, _ := cli_utils.RunCommand(t, "../mc cp a.txt custombucket", 1, time.Hour*2)

		assert.NotContains(t, output, "../mc: <ERROR>")

		os.Remove("a.txt") //nolint:errcheck
	})

	t.RunSequentially("Test for moving file", func(t *test.SystemTest) {
		_, _ = cli_utils.RunCommand(t, "../mc mb custombucket", 1, time.Hour*2)

		file, err := os.Create("a.txt")
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		defer file.Close() //nolint:errcheck

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
