package zsdk

import (
	"github.com/0chain/gosdk/zmagmacore/wallet"
)

type (
	info struct {
		logDir          string
		logLvl          string
		blockWorker     string
		signatureScheme string
	}
)

var (
	// Ensure Config implements interface.
	_ wallet.Config = (*info)(nil)
)

// NewInfo creates initialized info.
func NewInfo(logDir, logLvl, blockWorker, signatureScheme string) *info {
	return &info{
		logDir:          logDir,
		logLvl:          logLvl,
		blockWorker:     blockWorker,
		signatureScheme: signatureScheme,
	}
}

// LogDir implements wallet.Config interface.
func (i *info) LogDir() string {
	return i.logDir
}

// LogLvl implements wallet.Config interface.
func (i *info) LogLvl() string {
	return i.logLvl
}

// BlockWorker implements wallet.Config interface.
func (i *info) BlockWorker() string {
	return i.blockWorker
}

// SignatureScheme implements wallet.Config interface.
func (i *info) SignatureScheme() string {
	return i.signatureScheme
}
