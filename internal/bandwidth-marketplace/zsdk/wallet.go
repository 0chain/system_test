package zsdk

import (
	"path/filepath"

	"github.com/0chain/gosdk/zmagmacore/crypto"
	"github.com/0chain/gosdk/zmagmacore/node"
	"github.com/0chain/gosdk/zmagmacore/wallet"

	"github.com/0chain/system_test/internal/bandwidth-marketplace/config"
)

// Init initializes wallet and zsdk.
//
// Usually used as preparation to run transactions and zsdk interface.
func Init(keyPath, nodeDir, extID string, cfg *config.Config) error {
	// setup bandwidth-marketplace/code/core logger
	err := wallet.SetupZCNSDK(
		NewInfo(
			filepath.Join(nodeDir, "log", "it-zsdk.log"),
			"none",
			cfg.ServerChain.BlockWorker,
			cfg.ServerChain.SignatureScheme,
		),
	)
	if err != nil {
		return err
	}

	pbKey, prKey, err := crypto.ReadKeysFile(keyPath)
	if err != nil {
		return err
	}
	wall := wallet.New(pbKey, prKey)
	if err := wall.RegisterToMiners(); err != nil {
		return err
	}

	node.Start("", 0, extID, wall)

	return nil
}
