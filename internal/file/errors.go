package file

import (
	"errors"
	"fmt"
)

var (
	ErrFileExists = errors.New("file already exists")
)

type ErrDownloadFailed struct {
	Err error
}

func (e *ErrDownloadFailed) Error() string {
	return fmt.Errorf("failed to download file: %w", e.Err).Error()
}

type ErrPrepareFilepath struct {
	Err error
}

func (e *ErrPrepareFilepath) Error() string {
	return fmt.Errorf("failed to prepare file path: %w", e.Err).Error()
}

type ErrReadDir struct {
	Err error
}

func (e *ErrReadDir) Error() string {
	return fmt.Errorf("failed to read directory: %w", e.Err).Error()
}

type ErrOpenFile struct {
	Err error
}

func (e *ErrOpenFile) Error() string {
	return fmt.Errorf("failed to open file: %w", e.Err).Error()
}
