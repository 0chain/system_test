package cli_tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/assert"
)

func TestZs3ServerBucket(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	// test for moving the file from testbucket to testbucket2
	t.RunSequentially("Test for moving file from testbucket to testbucket2", func(t *test.SystemTest) {
		_, err := cli_utils.RunCommand(t, "../mc mb testbucket", 1, time.Hour*2)

		if err != nil {
			t.Fatalf("Error creating bucket: %v", err)
		}

		_, err = cli_utils.RunCommand(t, "../mc mb testbucket2", 1, time.Hour*2)
		if err != nil {
			t.Fatalf("Error creating bucket: %v", err)
		}

		file, err := os.Create("a.txt")
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		defer file.Close()

		_, err = file.WriteString("test")
		if err != nil {
			t.Fatalf("Error writing to file: %v", err)
		}

		output_ls, _ := cli_utils.RunCommand(t, "../mc mv a.txt testbucket", 1, time.Hour*2)

		assert.NotContains(t, output_ls, "../mc: <ERROR>")
		output, _ := cli_utils.RunCommand(t, "../mc mv testbucket/a.txt  testbucket2 ", 1, time.Hour*2)
		// output is in format ...sts/./mc_tests/testbucket/a.txt: 4 B / 4 B  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓  707 B/s 0s

		assert.NotContains(t, output, "../mc: <ERROR>")
		if _, err := os.Stat("testbucket2/a.txt"); err != nil {
			t.Errorf("Local file %s not found: %v", "testbucket2/a.txt", err)
		}

		// Assert that the local file exists
		assert.NoError(t, err, fmt.Sprintf("Local file %s should exist", "testbucket2/a.txt"))
	})
}
