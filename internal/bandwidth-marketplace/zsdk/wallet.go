package zsdk

import (
	"os"
	"path/filepath"

	"github.com/0chain/gosdk/core/zcncrypto"
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

// WriteDefaultKeysFile generates key pair with provided signature scheme and writes it to the provided path.
func WriteDefaultKeysFile(signatureScheme, path string) error {
	ss := zcncrypto.NewSignatureScheme(signatureScheme)
	if _, err := ss.GenerateKeys(); err != nil {
		return err
	}

	keys := ss.GetPublicKey() + "\n" + ss.GetPrivateKey()
	if err := os.WriteFile(path, []byte(keys), 0600); err != nil {
		return err
	}

	return nil
}
