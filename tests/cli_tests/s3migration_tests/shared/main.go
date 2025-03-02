package shared

import (
	"encoding/json"
	"fmt"
	"sync"

	util "github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/test"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	ConfigData          util.Config
	WalletMutex         sync.Mutex
	Wallets             []json.RawMessage
	WalletIdx           int64
	S3Client            *s3.S3
	ConfigDir           string
	ConfigPath          string
	RootPath            string
	DefaultAllocationId string
	DefaultWallet       string
)

const (
	KB         = 1024      // kilobyte
	MB         = 1024 * KB // megabyte
	GB         = 1024 * MB // gigabyte
	AllocSize  = int64(50 * MB)
	Chunksize  = 64 * 1024
	DirPrefix  = "dir"
	DirMaxRand = 1000
)

func SetupConfig(parsedConfig *util.Config) {
	ConfigData = *parsedConfig
}

func SetupAllocationWithWalletWithoutTest(t *test.SystemTest, walletName, cliConfigFilename string, extraParams ...map[string]interface{}) (string, error) {
	options := map[string]interface{}{"size": "10000000", "lock": "5"}

	for _, params := range extraParams {
		for k, v := range params {
			options[k] = v
		}
	}
	cli_utils.CreateWalletForName(RootPath, walletName)
	output, _ := cli_utils.CreateNewAllocationForWallet(t, walletName, cliConfigFilename, RootPath, cli_utils.CreateParams(options))
	defer func() {
		fmt.Printf("err: %v\n", output)
	}()
	return cli_utils.GetAllocationID(output[0])
}
