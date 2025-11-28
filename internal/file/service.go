package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/pkg/model"
	"slices"
	"strings"
	"sync"
)

type Service interface {
	DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error
	DeleteFolder(folderPath string) error
}

type DefaultService struct {
	downloader Downloader
	cfg        *config.FileServiceCfg
}

func NewDefaultService(downloader Downloader, cfg *config.FileServiceCfg) Service {
	return &DefaultService{
		downloader: downloader,
		cfg:        cfg,
	}
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error {
	wg := sync.WaitGroup{}
	errChan := make(chan FailedFile, len(files))
	// TODO: research optimal value and make it configurable
	sem := make(chan struct{}, 5)

	for _, file := range files {
		sem <- struct{}{}
		wg.Add(1)
		go d.processFile(ctx, folderPath, file, sem, &wg, errChan)
	}
	wg.Wait()
	close(errChan)

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

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file model.OrderFile, sem chan struct{}, wg *sync.WaitGroup, errChan chan FailedFile) {
	defer wg.Done()
	defer func() { <-sem }()
	filePath := filepath.Join(d.cfg.DirPath, folderPath, file.FileName)

	if file.TGFileID == nil {
		if file.FileBody == nil {
			errChan <- FailedFile{
				Filename: file.FileName,
				Err:      fmt.Errorf("unexpected empty file body for file with no telegram ID"),
			}
			return
		}

		fs := io.NopCloser(strings.NewReader(*file.FileBody))
		err := d.saveFile(filePath, fs)
		if err != nil {
			errChan <- FailedFile{
				Filename: file.FileName,
				Err:      err,
			}
		}
		return
	}

	fs, err := d.downloader.DownloadFile(ctx, *file.TGFileID)
	if err != nil {
		errChan <- FailedFile{
			Filename: file.FileName,
			Err:      err,
		}
		return
	}

	err = d.saveFile(filePath, fs)
	if err != nil {
		errChan <- FailedFile{
			Filename: file.FileName,
			Err:      err,
		}
	}
}

func (d *DefaultService) saveFile(filePath string, fs io.ReadCloser) error {
	defer closeFileStream()(fs)

	fileName := filepath.Base(filePath)
	isAppendFile := slices.Contains(d.cfg.AppendModeFilenames, fileName)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	if isAppendFile {
		out, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer closeFile()(out)

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

func (d *DefaultService) DeleteFolder(folderPath string) error {
	folderPath = filepath.Join(d.cfg.DirPath, folderPath)
	return os.RemoveAll(folderPath)
}
