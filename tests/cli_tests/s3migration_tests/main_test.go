package s3migration_tests

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/0chain/system_test/tests/cli_tests/s3migration_tests/shared"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
)

func setupDefaultConfig() {
	viper.SetDefault("nodes.miner01ID", "73ad5727612116c025bb4405bf3adb4a4a04867ae508c51cf885395bffc8a949")
	viper.SetDefault("nodes.miner02ID", "3ec9a42db3355f33c35750ce589ed717c08787997b7f34a7f1f9fb0a03f2b17c")
	viper.SetDefault("nodes.miner03ID", "c6f4b8ce5da386b278ba8c4e6cf98b24b32d15bc675b4d12c95e082079c91937")
	viper.SetDefault("nodes.sharder01ID", "ea26431f8adb7061766f1d6bbcc3b292d70dd59960d857f04b8a75e6a5bbe04f")
	viper.SetDefault("nodes.sharder02ID", "30001a01a888584772b7fee13934021ab8557e0ed471c0a3a454e9164180aef1")
}

func setupConfig() {
	setupDefaultConfig()

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	shared.ConfigPath = "config.yaml"
	shared.RootPath = filepath.Join(dir, "../")
	path := filepath.Join(dir, "../config")

	viper.SetConfigName("nodes")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	parsedConfig := config.Parse(filepath.Join(path, "cli_tests_config.yaml"))

	// Setup shared config with parsedConfig
	shared.SetupConfig(*parsedConfig)

	// Other logic...
}

func defaultData() {
	t := testing.T{}
	system_test := test.NewSystemTest(&t)

	defaultAllocationId := cli_utils.SetupAllocation(system_test, shared.ConfigData.ConnectionString, shared.RootPath, map[string]interface{}{
		"size": shared.AllocSize,
	})

	defaultWallet := "default"
	cli_utils.CreateWalletForName(shared.RootPath, defaultWallet)

	fmt.Fprintf(os.Stdout, "Default allocation ID: %s\n", defaultAllocationId)
	fmt.Fprintf(os.Stdout, "Default wallet: %s\n", defaultWallet)
}

func EscapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func GetConfigDir() string {
	var configDir string
	curr, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	configDir = filepath.Join(curr, "../config")
	return configDir
}

func TestMain(m *testing.M) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-2"), // Replace with your desired AWS region
		Credentials: credentials.NewStaticCredentials(shared.ConfigData.S3AccessKey, shared.ConfigData.S3SecretKey, ""),
	})
	if err != nil {
		log.Fatalln("Failed to create AWS session:", err)
		return
	}

	sess_dp, err_dp := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(shared.ConfigData.DropboxAccessToken, "", ""),
	})
	if err_dp != nil {
		log.Fatalln("Failed to create Dropbox session:", err_dp)
	}

	sess_gd, err_gd := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(shared.ConfigData.GdriveAccessToken, "", ""),
	})
	if err_gd != nil {
		log.Fatalln("Failed to create GDrive session:", err_gd)
	}

	cloudService := os.Getenv("CLOUD_SERVICE")

	if cloudService == "dropbox" {
		shared.S3Client = s3.New(sess_dp)
	} else if cloudService == "gdrive" {
		shared.S3Client = s3.New(sess_gd)
	} else {
		shared.S3Client = s3.New(sess)
	}

	shared.ConfigPath = os.Getenv("CONFIG_PATH")
	shared.ConfigDir = os.Getenv("CONFIG_DIR")
	shared.RootPath = os.Getenv("ROOT_PATH")

	if shared.ConfigDir == "" {
		shared.ConfigDir = GetConfigDir()
	}

	if shared.ConfigPath == "" {
		shared.ConfigPath = "./config.yaml"
	}

	shared.ConfigDir, _ = filepath.Abs(shared.ConfigDir)

	setupConfig()

	shared.WalletMutex.Lock()
	fileContent, err := os.ReadFile("../config/wallets/wallets.json")
	if err != nil {
		log.Println("Error reading file:", err)
		return
	}

	err = json.Unmarshal(fileContent, &shared.Wallets)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return
	}

	cliutils.SetWallets(shared.Wallets)
	shared.WalletIdx = 500
	shared.WalletMutex.Unlock()

	defaultData()

	// exitRun := m.Run()
	// os.Exit(exitRun)
}
