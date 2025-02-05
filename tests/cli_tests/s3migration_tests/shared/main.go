package shared

import (
	"encoding/json"
	"sync"

	util "github.com/0chain/system_test/internal/api/util/config"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	ConfigData  util.Config
	WalletMutex sync.Mutex
	Wallets     []json.RawMessage
	WalletIdx   int64
	S3Client    *s3.S3
	ConfigDir   string
	ConfigPath  string
	RootPath    string
	DefaultAllocationId string
	DefaultWallet  string
)

const (
	KB        = 1024      // kilobyte
	MB        = 1024 * KB // megabyte
	GB        = 1024 * MB // gigabyte
	AllocSize = int64(50 * MB)
)

func SetupConfig(parsedConfig util.Config) {
	ConfigData = parsedConfig
}
