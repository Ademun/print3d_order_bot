package file

import (
	"context"
	"fmt"
	"io"
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
	defer fs.Close()
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, fs)

	return err
}
