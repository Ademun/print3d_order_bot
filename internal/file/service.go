package file

import (
	"context"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/pkg/model"
	"sync"

	"go.uber.org/atomic"
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

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []model.File) chan DownloadResult {
	wg := sync.WaitGroup{}
	counter := atomic.NewInt32(0)
	result := make(chan DownloadResult)
	// TODO: research optimal value and make it configurable
	sem := make(chan struct{}, 5)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for _, file := range files {
			sem <- struct{}{}
			wg.Add(1)
			go func(f model.File) {
				defer func() {
					<-sem
					wg.Done()
				}()
				d.processFile(ctx, folderPath, f, len(files), counter, result)
			}(file)
		}
		wg.Wait()
		close(result)
	}()

	return result
}

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file model.File, total int, counter *atomic.Int32, result chan DownloadResult) {
	filePath := filepath.Join(d.cfg.DirPath, folderPath, file.Name)

	if file.TGFileID == nil {
		result <- DownloadResult{
			Result: nil,
			Err:    ErrNoTgFileID,
		}
		return
	}

	dst, err := prepareFilepath(filePath)
	if err != nil {
		result <- DownloadResult{
			Result: nil,
			Err:    &ErrPrepareFilepath{Err: err},
		}
		return
	}

	err = d.downloader.DownloadFile(ctx, *file.TGFileID, dst)
	if err != nil {
		if err := os.Remove(filePath); err != nil {
			result <- DownloadResult{
				Result: nil,
				Err:    &ErrDownloadFailed{Err: err},
			}
		}
		result <- DownloadResult{
			Result: nil,
			Err:    &ErrDownloadFailed{Err: err},
		}
	}

	checksum, err := calculateChecksum(filePath)
	if err != nil {
		result <- DownloadResult{
			Result: nil,
			Err:    ErrCalculateChecksum,
		}
	}

	counter.Inc()
	result <- DownloadResult{
		Result: &model.File{
			Name:     file.Name,
			Checksum: checksum,
			TGFileID: file.TGFileID,
		},
		Index: int(counter.Load()),
		Total: total,
		Err:   nil,
	}
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
