package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/pkg/model"
	"sync"
)

type Service interface {
	CreateFolder(folderPath string) error
	DownloadAndSave(ctx context.Context, folderPath string, files []model.File) error
	GetFiles(ctx context.Context, folderPath string) (chan model.FileResult, error)
	DeleteFolder(folderPath string) error
	SetDownloader(downloader Downloader)
}

type DefaultService struct {
	downloader Downloader
	cfg        *config.FileServiceCfg
	wg         sync.WaitGroup
}

func NewDefaultService(downloader Downloader, cfg *config.FileServiceCfg) Service {
	return &DefaultService{
		downloader: downloader,
		cfg:        cfg,
	}
}

func (d *DefaultService) CreateFolder(folderPath string) error {
	path := filepath.Join(d.cfg.DirPath, folderPath)
	return os.MkdirAll(path, os.ModePerm)
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []model.File) error {
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

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file model.File, sem chan struct{}, wg *sync.WaitGroup, errChan chan FailedFile) {
	defer wg.Done()
	defer func() { <-sem }()
	filePath := filepath.Join(d.cfg.DirPath, folderPath, file.Name)

	dst, err := d.prepareFilepath(filePath)
	if err != nil {
		errChan <- FailedFile{
			Filename: file.Name,
			Err:      err,
		}
	}

	if file.TGFileID == nil {
		errChan <- FailedFile{
			Filename: file.Name,
			Err:      fmt.Errorf("file %s has no TGFileID", file.Name),
		}
		return
	}

	err = d.downloader.DownloadFile(ctx, *file.TGFileID, dst)
	if err != nil {
		errChan <- FailedFile{
			Filename: file.Name,
			Err:      err,
		}
		return
	}
}

func (d *DefaultService) prepareFilepath(filePath string) (io.Writer, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		out, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}

		return out, err
	}

	return nil, fmt.Errorf("file already exists: %s", filePath)
}

func (d *DefaultService) GetFiles(ctx context.Context, folderPath string) (chan model.FileResult, error) {
	d.wg.Add(1)
	defer d.wg.Done()

	path := filepath.Join(d.cfg.DirPath, folderPath)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make(chan model.FileResult)
	wg := sync.WaitGroup{}

	go func() {
		for _, entry := range entries {
			wg.Add(1)
			go func(entry os.DirEntry) {
				defer wg.Done()

				if entry.IsDir() {
					return
				}

				file, err := os.Open(filepath.Join(path, entry.Name()))

				files <- model.FileResult{
					Filename: entry.Name(),
					File:     file,
					Err:      err,
				}
			}(entry)
		}
		wg.Wait()
		close(files)
	}()

	return files, nil
}

func (d *DefaultService) DeleteFolder(folderPath string) error {
	folderPath = filepath.Join(d.cfg.DirPath, folderPath)
	return os.RemoveAll(folderPath)
}

func (d *DefaultService) SetDownloader(downloader Downloader) {
	d.downloader = downloader
}
