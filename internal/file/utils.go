package file

import (
	"errors"
	"io"
	"os"
	"path/filepath"
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
