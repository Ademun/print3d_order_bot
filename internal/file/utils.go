package file

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
)

func prepareFilepath(filePath string) (io.Writer, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(filePath); errors.Is(os.ErrExist, err) {
		return nil, ErrFileExists
	}

	out, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func calculateChecksum(filePath string) (uint64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	hasher := xxhash.New()
	if _, err = io.Copy(hasher, file); err != nil {
		return 0, err
	}

	return hasher.Sum64(), nil
}
