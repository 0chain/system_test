package setup

import (
	"github.com/0chain/system_test/internal/api/util/config"
	"github.com/0chain/system_test/internal/api/util/tx_client"
)

// SetupTestEnvironment performs initial setup for test environment
func InitTestEnvironment(configPath string) (apiClient *client.Client, txClient *tx_client.Client) {
	parsedConfig := config.Parse(configPath)

	apiClient = client.NewClient(parsedConfig.NetworkEntrypoint)
	txClient = tx_client.NewClient()

	return
}
