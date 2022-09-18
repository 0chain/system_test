package interaction

import (
	"crypto/rand"
	"os"
)

// CreateFile creates a new file and fills it with random data
func CreateFile(name string, size int64) (*os.File, error) {
	buffer := make([]byte, size)
	_, err := rand.Read(buffer)
	if err != nil {
		return nil, err
	} //nolint:gosec,revive

	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(buffer)
	if err != nil {
		return nil, err
	}
	return file, nil
}
