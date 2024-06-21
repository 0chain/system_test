package zs3servertests

import (
	"log"
	"strings"
	"testing"
	"time"

	test "github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestZs3serverWarpTests(testSetup *testing.T) {
	log.Println("Running Warp List Benchmark...")
	t := test.NewSystemTest(testSetup)
	server, host, accessKey, secretKey, _ := cliutils.ReadFile(testSetup)
	commandGenerated := "../warp get --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + " --duration 30s" + " --obj.size 1KiB"
	log.Println("Command Generated: ", commandGenerated)
	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
	if err != nil {
		testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
	}
	log.Println("Warp List Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]

	output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

	output_string = "Condition 1: Get objects: 1 \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-list_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}

func TestZs3serverWarpConcurrentTests(testSetup *testing.T) {
	log.Println("Running Warp List Benchmark with concurrent ...")
	t := test.NewSystemTest(testSetup)
	server, host, accessKey, secretKey, concurrent := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp get --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s" + " --obj.size 1KiB"
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

	if err != nil {
		testSetup.Fatalf("Error running warp list: %v\nOutput: %s", err, output)
	}
	log.Println("Warp List Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]

	output_string = "Condition 1: Get objects: 100 concurrent 50::  \n--------\n" + output_string
	log.Println("APending to file with this stat ", output_string)

	err = cliutils.AppendToFile("warp-list_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}

func TestZs3serverPutWarpTests(testSetup *testing.T) {
	log.Println("Running Warp Put Benchmark...")
	t := test.NewSystemTest(testSetup)

	server, host, accessKey, secretKey, concurrent := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp put --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s" + " --obj.size 1KiB"
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

	if err != nil {
		testSetup.Fatalf("Error running warp put: %v\nOutput: %s", err, output)
	}
	log.Println("Warp Put Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]

	output_string = "Condition 2 : Put  \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}

func TestZs3serverRetentionTests(testSetup *testing.T) {
	log.Println("Running Warp Retention Benchmark...")
	t := test.NewSystemTest(testSetup)

	server, host, accessKey, secretKey, concurrent := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp retention --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s" + " --obj.size 1KiB"
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)

	if err != nil {
		testSetup.Fatalf("Error running warp retention: %v\nOutput: %s", err, output)
	}
	log.Println("Warp Retention Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]
	output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

	output_string = "Condition 1: Retention : objects: 1 \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}

func TestZs3serverMultipartTests(testSetup *testing.T) {
	log.Println("Running Warp Multipart Benchmark...")
	t := test.NewSystemTest(testSetup)

	server, host, accessKey, secretKey, concurrent := cliutils.ReadFile(testSetup)

	commandGenerated := "../warp multipart --parts=500 --part.size=10MiB --host=" + server + ":" + host + " --access-key=" + accessKey + " --secret-key=" + secretKey + "  --concurrent " + concurrent + " --duration 30s"
	log.Println("Command Generated: ", commandGenerated)

	output, err := cliutils.RunCommand(t, commandGenerated, 1, time.Hour*2)
	if err != nil {
		testSetup.Fatalf("Error running warp multipart: %v\nOutput: %s", err, output)
	}
	log.Println("Warp Multipart Output:\n", output)
	output_string := strings.Join(output, "\n")
	output_string = strings.Split(output_string, "----------------------------------------")[1]
	output_string = strings.Split(output_string, "warp: Starting cleanup")[0]

	output_string = "Condition 1: Retention : objects: 1 \n--------\n" + output_string
	err = cliutils.AppendToFile("warp-put_output.txt", output_string)

	if err != nil {
		testSetup.Fatalf("Error appending to file: %v\n", err)
	}
}
