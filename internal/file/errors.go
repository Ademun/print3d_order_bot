package file

import (
	"errors"
	"fmt"
)

var (
	ErrNoTgFileID        = errors.New("file does not have telegram file_id")
	ErrFileExists        = errors.New("file already exists")
	ErrCalculateChecksum = errors.New("failed to calculate checksum")
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
