package cli_tests

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
)

func read_file_mc(testSetup *testing.T) (string, string, string, string, string, string, string, bool) {
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
	use_command, ok := hosts["use_command"].(bool)

	if !ok {
		use_command = false
	}

	host := strconv.FormatInt(int64(port), 10)
	secondary_port := strconv.FormatInt(int64(s_port), 10)
	concurrent_no := strconv.FormatInt(int64(concurrent), 10)
	return server, host, accessKey, secretKey, concurrent_no, secondary_port, secondary_server,use_command

}

func splitCmdString(cmdString string) ([]string, error) {
    return []string{"sh", "-c", cmdString}, nil
}

func logOutput(stdout  io.Reader, t *test.SystemTest) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		t.Logf("[MinIO stdout] %s", scanner.Text())
	}
}

func killProcess(port string) (int, error) {
    cmd := exec.Command("lsof", "-t", "-i", fmt.Sprintf(":%s", port))
    out, err := cmd.Output()
    if err != nil {
        return 0, fmt.Errorf("error running lsof -i command: %v", err)
    }
    pidStr := strings.TrimSpace(string(out))
    if pidStr == "" {
        return 0, fmt.Errorf("no process found for port %s", port)
    }
    pid, err := strconv.Atoi(pidStr)
    if err != nil {
        return 0, fmt.Errorf("error converting PID to integer: %v", err)
    }
	// killing process by id
	cmd = exec.Command("kill", strconv.Itoa(pid))

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("Failed to kill process with PID %d: %v\n", pid, err)
	}

    return pid, nil
}

func TestZs3ServerReplication(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	server, port, accessKey, secretKey, _, s_port, s_server, use_command := read_file_mc(testSetup)
	log.Print(s_port)

	currentUser, err := user.Current()
    if err != nil {
        panic(err)
    }
	var cmd *exec.Cmd
	if use_command{
		zcnDir := filepath.Join(currentUser.HomeDir, ".zcn")

		cmdString := "export MINIO_ROOT_USER="+accessKey+" && export MINIO_ROOT_PASSWORD="+secretKey+" && ../minio gateway zcn --configDir "+zcnDir + " --console-address :8000"

		cmdParts, err := splitCmdString(cmdString)
		if err != nil {
			fmt.Println("Error splitting command string:", err)
			os.Exit(1)
		}
		cmd = exec.Command(cmdParts[0], cmdParts[1:]...)

		_, err = cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("Error creating stdout pipe: %v", err)
		}

		_, _ = cmd.StderrPipe()


		log.Printf("Generated command: %s %s", cmd.Path, cmd.Args)

		err = cmd.Start()
		if err != nil {
			fmt.Println("Error starting MinIO server:", err)
			os.Exit(1)
		}
		// go logOutput(stdout, t)
		// go logOutput(stderr, t)
		time.Sleep(5 *time.Second)
		t.Logf("MinIO server started successfully")
	}

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

		_, _ =  cli_utils.RunCommand(t, "../mc mirror --overwrite primary/mybucket secondary/mirrorbucket ", 1, time.Minute*2)

		t.Log("removing... the a.txt from primary bucket")
		_, _ = cli_utils.RunCommand(t, "../mc rm primary/mybucket/a.txt", 1, time.Second*2)
		t.Log("listing... secondary bucket")

		output, err  :=cli_utils.RunCommand(t, "../mc ls secondary/mirrorbucket", 1, time.Second*2)
		if err != nil {
			t.Log(err, "err of command")
		}

		t.Log("All operations are completed")
		t.Log("Cleaning up ..... ")
		_, _= cli_utils.RunCommand(t, "../mc rm primary/mybucket", 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t, "../mc rm secondary/mirrorbucket", 1, time.Hour*2)

		_, _ = cli_utils.RunCommand(t, "../mc alias rm primary", 1, 2*time.Hour)
    	_, _ = cli_utils.RunCommand(t, "../mc alias rm secondary", 1, 2*time.Hour)
		_ = os.Remove("a.txt")

		assert.Contains(t, strings.Join(output, "\n"), "a.txt")
		_, err = killProcess("9000")

		if err != nil {
			t.Logf("Error killing the command process")
		}
	})
}
