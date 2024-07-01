package cli_tests

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
)

func read_file_mc(testSetup *testing.T) (string, string, string, string, string, string, string) {
	file, err := os.Open("mc_hosts.yaml")
	if err != nil {
		testSetup.Fatalf("Error opening hosts.yaml file: %v\n", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var hosts map[string]interface{}
	err = decoder.Decode(&hosts)
	if err != nil {
		testSetup.Fatalf("Error decoding mc_hosts.yaml file: %v\n", err)
	}

	accessKey := hosts["access_key"].(string)
	secretKey := hosts["secret_key"].(string)
	port := hosts["port"].(int)
	concurrent := hosts["concurrent"].(int)
	server := hosts["server"].(string)
	secondary_server := hosts["secondary_server"].(string)
	s_port := hosts["secondary_port"].(int)

	host := strconv.FormatInt(int64(port), 10)
	secondary_port := strconv.FormatInt(int64(s_port), 10)
	concurrent_no := strconv.FormatInt(int64(concurrent), 10)
	return server, host, accessKey, secretKey, concurrent_no, secondary_port, secondary_server

}

func TestZs3ServerReplication(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	server, port, accessKey, secretKey, _, s_port, s_server := read_file_mc(testSetup)
	// run minio server
	log.Print(s_port)
	cmd := exec.Command("../minio", "gateway", "zcn", "--console-address", ":8000")

	err := cmd.Start()
	if err != nil {
		fmt.Println(err)
	}
	t.Logf("Minio server startted")

	t.RunWithTimeout("Test for replication",4000 * time.Second, func(t *test.SystemTest) {
		t.Log(server, "server")
		command_primary := "../mc alias set primary http://"+server+":"+port+" "+accessKey+" "+secretKey+" --api S3v2"
		t.Log(command_primary, "command Generated")

		command_secondary := "../mc alias set secondary http://"+s_server+":"+port+" "+accessKey+" "+secretKey+" --api S3v2"
		t.Log(command_secondary, "command Generated")

		_, _ = cli_utils.RunCommand(t, command_primary, 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t,command_secondary , 1, time.Hour*2)

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

		_, _ = cli_utils.RunCommand(t, "../mc mb secondary/mirrorbucket", 1, time.Hour * 2)

		t.Log("copying... the a.txt")
		_, _ = cli_utils.RunCommand(t, "../mc cp a.txt primary/mybucket", 1, time.Second*2)

		cmd := exec.Command("../mc", "mirror", "--watch", "--overwrite", "primary/mybucket", "secondary/mirrorbucket", "--remove", "--watch")

		if err := cmd.Start(); err != nil {
			log.Fatalf("Failed to start mirroring: %v", err)
		}

		t.Log("removing... the a.txt from primary bucket")
		_, _ = cli_utils.RunCommand(t, "../mc rm primary/mybucket/a.txt", 1, time.Second*2)
		t.Log("listing... primary bucket")

		output, err  :=cli_utils.RunCommand(t, "../mc ls primary/mybucket", 1, time.Second*2)
		t.Log(output, "output of command")
		t.Log(err, "err of command")

		assert.Contains(t, output, "a.txt")

		t.Log("All operations are completed")
		t.Log("Cleaning up ..... ")
		_, _= cli_utils.RunCommand(t, "../mc rm primary/mybucket", 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t, "../mc rm secondary/mirrorbucket", 1, time.Hour*2)

		_, _ = cli_utils.RunCommand(t, "../mc alias rm primary", 1, 2*time.Hour)
    	_, _ = cli_utils.RunCommand(t, "../mc alias rm secondary", 1, 2*time.Hour)
	})
}
