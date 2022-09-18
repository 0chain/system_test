package setup

import (
	"github.com/0chain/system_test/internal/api/util/client"
	"github.com/0chain/system_test/internal/api/util/config"
)

// InitTestEnvironment performs initial setup for test environment
func InitTestEnvironment(configPath string) (apiClient *client.APIClient, sdkClient *client.SDKClient) {
	parsedConfig := config.Parse(configPath)

	sdkClient = client.NewSDKClient(parsedConfig.NetworkEntrypoint)
	apiClient = client.NewAPIClient(parsedConfig.NetworkEntrypoint)
	return
}
