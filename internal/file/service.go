package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/pkg/model"
	"strings"
	"sync"
)

type Service interface {
	DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error
}

type DefaultService struct {
	downloader Downloader
}

func NewDefaultService(downloader Downloader) Service {
	return &DefaultService{downloader: downloader}
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error {
	wg := sync.WaitGroup{}
	errChan := make(chan FailedFile, len(files))
	// TODO: research optimal value and make it configurable
	sem := make(chan struct{}, 5)

	for _, file := range files {
		sem <- struct{}{}
		wg.Add(1)
		go d.processFile(ctx, folderPath, file, &wg, errChan)
	}
	wg.Wait()

	var failedFiles []FailedFile
	for ff := range errChan {
		failedFiles = append(failedFiles, ff)
	}

	if len(failedFiles) > 0 {
		return &ErrProcessingFiles{
			TotalFiles:  len(files),
			FailedFiles: failedFiles,
		}
	}

	return nil
}

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file model.OrderFile, wg *sync.WaitGroup, errChan chan FailedFile) {
	defer wg.Done()
	select {
	case <-ctx.Done():
		errChan <- FailedFile{
			Filename: file.FileName,
			Err:      ctx.Err(),
		}
	default:
		filePath := filepath.Join(folderPath, file.FileName)

		if file.TGFileID == nil {
			if file.FileBody == nil {
				errChan <- FailedFile{
					Filename: file.FileName,
					Err:      fmt.Errorf("unexpected empty file body for file with no telegram ID"),
				}
			}

			fs := io.NopCloser(strings.NewReader(*file.FileBody))
			err := d.saveFile(filePath, fs)
			if err != nil {
				errChan <- FailedFile{
					Filename: file.FileName,
					Err:      err,
				}
			}
		}

		fs, err := d.downloader.DownloadFile(ctx, *file.TGFileID)
		if err != nil {
			errChan <- FailedFile{
				Filename: file.FileName,
				Err:      err,
			}
		}

		err = d.saveFile(filePath, fs)
		if err != nil {
			errChan <- FailedFile{
				Filename: file.FileName,
				Err:      err,
			}
		}
	}
}

func (d *DefaultService) saveFile(filePath string, fs io.ReadCloser) error {
	defer closeFileStream()(fs)

	fileName := filepath.Base(filePath)
	isSpecialFile := fileName == "links.txt" || fileName == "comments.txt"

	if isSpecialFile {
		out, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer closeFile()

		_, err = io.Copy(out, fs)
		return err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		out, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer closeFile()(out)

		_, err = io.Copy(out, fs)
		return err
	}

	return fmt.Errorf("file already exists: %s", filePath)
}

func closeFileStream() func(fs io.ReadCloser) {
	return func(fs io.ReadCloser) {
		if err := fs.Close(); err != nil {
			slog.Error("failed to close file stream", "err", err)
		}
	}
}

func closeFile() func(out *os.File) {
	return func(out *os.File) {
		if err := out.Close(); err != nil {
			slog.Error("failed to close file", "err", err)
		}
	}
}
