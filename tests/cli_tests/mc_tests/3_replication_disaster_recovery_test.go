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

		// Remove aliases if they exist (ignore errors)
		_, _ = cli_utils.RunCommand(t, "../mc alias rm primary", 1, time.Second*5)
		_, _ = cli_utils.RunCommand(t, "../mc alias rm secondary", 1, time.Second*5)

		command_primary := "../mc alias set primary http://" + config.Server + ":" + config.HostPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
		t.Log(command_primary, "command Generated")

		command_secondary := "../mc alias set secondary http://" + config.SecondaryServer + ":" + config.SecondaryPort + " " + config.AccessKey + " " + config.SecretKey + " --api S3v2"
		t.Log(command_secondary, "command Generated")

		output, err := cli_utils.RunCommand(t, command_primary, 1, time.Hour*2)
		if err != nil {
			t.Fatalf("Failed to set primary mc alias: %v\nOutput: %s", err, output)
		}

		output, err = cli_utils.RunCommand(t, command_secondary, 1, time.Hour*2)
		if err != nil {
			t.Fatalf("Failed to set secondary mc alias: %v\nOutput: %s", err, output)
		}

		// Verify aliases were set correctly by listing them
		aliasListOutput, aliasListErr := cli_utils.RunCommand(t, "../mc alias list", 1, time.Minute*2)
		if aliasListErr != nil {
			t.Fatalf("Failed to list mc aliases: %v\nOutput: %s", aliasListErr, aliasListOutput)
		}
		aliasListStr := strings.Join(aliasListOutput, "\n")
		if !strings.Contains(aliasListStr, "primary") {
			t.Fatalf("Alias 'primary' was not found in alias list. Output: %s", aliasListStr)
		}
		if !strings.Contains(aliasListStr, "secondary") {
			t.Fatalf("Alias 'secondary' was not found in alias list. Output: %s", aliasListStr)
		}

		// Create bucket, with retry logic if alias issue detected
		bucketCommand := "../mc mb primary/mybucket"
		bucketOutput, bucketErr := cli_utils.RunCommand(t, bucketCommand, 1, time.Hour*2)
		if bucketErr != nil {
			outputStr := strings.Join(bucketOutput, "\n")
			if strings.Contains(outputStr, "already exists") || strings.Contains(outputStr, "BucketAlreadyExists") {
				// Bucket already exists, which is fine - continue
				t.Logf("Primary bucket already exists, continuing...")
			} else if strings.Contains(outputStr, "does not exist") {
				// Alias might not be properly configured, try to recreate it
				t.Logf("Primary alias issue detected, retrying alias setup...")
				output, err = cli_utils.RunCommand(t, command_primary, 1, time.Hour*2)
				if err != nil {
					t.Fatalf("Failed to recreate primary mc alias: %v\nOutput: %s", err, output)
				}
				// Verify alias again after recreation
				aliasListOutput, aliasListErr = cli_utils.RunCommand(t, "../mc alias list", 1, time.Minute*2)
				if aliasListErr == nil {
					aliasListStr = strings.Join(aliasListOutput, "\n")
					if !strings.Contains(aliasListStr, "primary") {
						t.Fatalf("Alias 'primary' still not found after retry. Output: %s", aliasListStr)
					}
				}
				// Retry bucket creation
				bucketOutput, bucketErr = cli_utils.RunCommand(t, bucketCommand, 1, time.Hour*2)
				if bucketErr != nil {
					outputStr = strings.Join(bucketOutput, "\n")
					if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "BucketAlreadyExists") {
						t.Fatalf("Failed to create primary bucket after retry: %v\nOutput: %s", bucketErr, outputStr)
					}
				} else {
					t.Logf("Primary bucket created successfully after retry")
				}
			} else {
				t.Fatalf("Failed to create primary bucket: %v\nOutput: %s", bucketErr, outputStr)
			}
		} else {
			t.Logf("Primary bucket created successfully")
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

		// Create bucket, with retry logic if alias issue detected
		bucketCommand2 := "../mc mb secondary/mirrorbucket"
		bucketOutput2, bucketErr2 := cli_utils.RunCommand(t, bucketCommand2, 1, time.Hour*2)
		if bucketErr2 != nil {
			outputStr := strings.Join(bucketOutput2, "\n")
			if strings.Contains(outputStr, "already exists") || strings.Contains(outputStr, "BucketAlreadyExists") {
				// Bucket already exists, which is fine - continue
				t.Logf("Secondary bucket already exists, continuing...")
			} else if strings.Contains(outputStr, "does not exist") {
				// Alias might not be properly configured, try to recreate it
				t.Logf("Secondary alias issue detected, retrying alias setup...")
				output, err = cli_utils.RunCommand(t, command_secondary, 1, time.Hour*2)
				if err != nil {
					t.Fatalf("Failed to recreate secondary mc alias: %v\nOutput: %s", err, output)
				}
				// Verify alias again after recreation
				aliasListOutput, aliasListErr = cli_utils.RunCommand(t, "../mc alias list", 1, time.Minute*2)
				if aliasListErr == nil {
					aliasListStr = strings.Join(aliasListOutput, "\n")
					if !strings.Contains(aliasListStr, "secondary") {
						t.Fatalf("Alias 'secondary' still not found after retry. Output: %s", aliasListStr)
					}
				}
				// Retry bucket creation
				bucketOutput2, bucketErr2 = cli_utils.RunCommand(t, bucketCommand2, 1, time.Hour*2)
				if bucketErr2 != nil {
					outputStr = strings.Join(bucketOutput2, "\n")
					if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "BucketAlreadyExists") {
						t.Fatalf("Failed to create secondary bucket after retry: %v\nOutput: %s", bucketErr2, outputStr)
					}
				} else {
					t.Logf("Secondary bucket created successfully after retry")
				}
			} else {
				t.Fatalf("Failed to create secondary bucket: %v\nOutput: %s", bucketErr2, outputStr)
			}
		} else {
			t.Logf("Secondary bucket created successfully")
		}

		t.Log("copying... the a.txt")
		_, _ = cli_utils.RunCommand(t, "../mc cp a.txt primary/mybucket", 1, time.Second*2)

		_, _ = cli_utils.RunCommand(t, "../mc mirror --overwrite primary/mybucket secondary/mirrorbucket ", 1, time.Minute*2)

		t.Log("removing... the a.txt from primary bucket")
		_, _ = cli_utils.RunCommand(t, "../mc rm primary/mybucket/a.txt", 1, time.Second*2)
		t.Log("listing... secondary bucket")

		lsOutput, lsErr := cli_utils.RunCommand(t, "../mc ls secondary/mirrorbucket", 1, time.Second*2)
		if lsErr != nil {
			t.Log(lsErr, "err of command")
		}

		t.Log("All operations are completed")
		t.Log("Cleaning up ..... ")

		// Remove buckets if they exist (ignore errors if they don't exist)
		_, _ = cli_utils.RunCommand(t, "../mc rm --recursive --force primary/mybucket", 1, time.Hour*2)
		_, _ = cli_utils.RunCommand(t, "../mc rm --recursive --force secondary/mirrorbucket", 1, time.Hour*2)

		// Remove aliases if they exist (ignore errors if they don't exist)
		_, _ = cli_utils.RunCommand(t, "../mc alias rm primary", 1, 2*time.Hour)
		_, _ = cli_utils.RunCommand(t, "../mc alias rm secondary", 1, 2*time.Hour)
		_ = os.Remove("a.txt")

		assert.Contains(t, strings.Join(lsOutput, "\n"), "a.txt")
		_, err = cli_utils.KillProcess()

		if err != nil {
			t.Logf("Error killing the command process")
		}
	})
}
