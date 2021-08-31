package docker

import (
	"os"
)

const (
	// magmaLogDir represents directory where magmas logs are stored.
	magmaLogDir = "./src/magma/log"

	// consumerLogDir represents directory where consumers logs are stored.
	consumerLogDir = "./src/consumer/log"

	// consumerDataDir represents directory where consumers data are stored.
	consumerDataDir = "./src/consumer/data"

	// providerLogDir represents directory where providers logs are stored.
	providerLogDir = "./src/provider/log"

	// providerLogDir represents directory where providers logs are stored.
	providerDataDir = "./src/provider/data"
)

// InitDirs initializes logs and data directories of magma, consumer and provider.
func InitDirs() error {
	if err := os.MkdirAll(magmaLogDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(consumerLogDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(providerLogDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(consumerDataDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(providerDataDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

// CleanDirs cleans logs and data directories of magma, consumer and provider.
func CleanDirs() error {
	if err := os.RemoveAll(magmaLogDir); err != nil {
		return err
	}

	if err := os.RemoveAll(consumerLogDir); err != nil {
		return err
	}

	if err := os.RemoveAll(providerLogDir); err != nil {
		return err
	}

	if err := os.RemoveAll(consumerDataDir); err != nil {
		return err
	}

	if err := os.RemoveAll(providerDataDir); err != nil {
		return err
	}

	return nil
}
