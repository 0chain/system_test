package cli_tests

import (
	"os"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
)

func TestZs3ServerReplication(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	config := cli_utils.ReadFileMC(testSetup)

	t.RunWithTimeout("Test for replication", 4000*time.Second, func(t *test.SystemTest) {
		t.Log(config.Server, "server")
		command_primary := "../mc alias set primary http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
		t.Log(command_primary, "command Generated")

		command_secondary := "../mc alias set secondary http://" + config.SecondaryServer + ":" + config.SecondaryPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
		t.Log(command_secondary, "command Generated")

		_, _ = cli_utils.RunCommand(t, command_primary, 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t, command_secondary, 1, time.Hour*2)

		_, _ = cli_utils.RunCommand(t, "../mc mb primary/mybucket", 1, time.Hour*2)

		file, err := os.Create("a.txt")
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		defer file.Close()

		_, err = file.WriteString("test")
		if err != nil {
			t.Fatalf("Error writing to file: %v", err)
		}

		_, _ = cli_utils.RunCommand(t, "../mc mb secondary/mirrorbucket", 1, time.Hour*2)

		t.Log("copying... the a.txt")
		_, _ = cli_utils.RunCommand(t, "../mc cp a.txt primary/mybucket", 1, time.Second*2)

		_, _ = cli_utils.RunCommand(t, "../mc mirror --overwrite primary/mybucket secondary/mirrorbucket ", 1, time.Minute*2)

		t.Log("removing... the a.txt from primary bucket")
		_, _ = cli_utils.RunCommand(t, "../mc rm primary/mybucket/a.txt", 1, time.Second*2)
		t.Log("listing... secondary bucket")

		output, err := cli_utils.RunCommand(t, "../mc ls secondary/mirrorbucket", 1, time.Second*2)
		if err != nil {
			t.Log(err, "err of command")
		}

		t.Log("All operations are completed")
		t.Log("Cleaning up ..... ")
		_, _ = cli_utils.RunCommand(t, "../mc rm primary/mybucket", 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t, "../mc rm secondary/mirrorbucket", 1, time.Hour*2)

		_, _ = cli_utils.RunCommand(t, "../mc alias rm primary", 1, 2*time.Hour)
		_, _ = cli_utils.RunCommand(t, "../mc alias rm secondary", 1, 2*time.Hour)
		_ = os.Remove("a.txt")

		assert.Contains(t, strings.Join(output, "\n"), "a.txt")
		_, err = cli_utils.KillProcess()

		if err != nil {
			t.Logf("Error killing the command process")
		}
	})
}
