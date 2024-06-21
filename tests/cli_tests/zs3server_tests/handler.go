package zs3servertests

import (
	"log"
	"os"
	"strconv"
	"testing"

	"gopkg.in/yaml.v2"
)

func read_file(testSetup *testing.T) (string, string, string, string, string) {
	file, err := os.Open("hosts.yaml")
	if err != nil {
		testSetup.Fatalf("Error opening hosts.yaml file: %v\n", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var hosts map[string]interface{}
	err = decoder.Decode(&hosts)
	if err != nil {
		testSetup.Fatalf("Error decoding hosts.yaml file: %v\n", err)
	}

	accessKey := hosts["access_key"].(string)
	secretKey := hosts["secret_key"].(string)
	port := hosts["port"].(int)
	concurrent := hosts["concurrent"].(int)
	server := hosts["server"].(string)
	host := strconv.FormatInt(int64(port), 10)
	concurrent_no := strconv.FormatInt(int64(concurrent), 10)
	return server, host, accessKey, secretKey, concurrent_no

}

func read_file_allocation() (string, string, string) {
	file, err := os.Open("allocation.yaml")
	if err != nil {
		log.Fatalln("Error reading the file:", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var allocation_data map[string]interface{}
	err = decoder.Decode(&allocation_data)
	if err != nil {
		log.Fatal("Error decoding allocation.yaml file:", err)
	}
	data := allocation_data["data"].(int)
	parity := allocation_data["parity"].(int)
	lock := allocation_data["lock"].(int)

	data_str := strconv.FormatInt(int64(data), 10)
	parity_str := strconv.FormatInt(int64(parity), 10)
	lock_str := strconv.FormatInt(int64(lock), 10)
	return data_str, parity_str, lock_str
}

func appendToFile(filename string, data string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(data); err != nil {
		return err
	}
	return nil
}

// func testforErrors(cmd *exec.Cmd) {
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	zlogger.Logger.Info(string(out))
// }
