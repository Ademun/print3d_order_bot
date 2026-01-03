package file

import (
	"context"
	"os"
	"path/filepath"
	"print3d-order-bot/pkg/config"
	"sync"

	"go.uber.org/atomic"
)

type Service interface {
	CreateFolder(folderPath string) error
	DownloadAndSave(ctx context.Context, folderPath string, files []RequestFile, downloader Downloader) chan DownloadResult
	ReadFiles(folderPath string) (chan ReadResult, error)
	DeleteFolder(folderPath string) error
}

type DefaultService struct {
	cfg *config.FileServiceCfg
	wg  sync.WaitGroup
}

func NewDefaultService(cfg *config.FileServiceCfg) Service {
	return &DefaultService{
		cfg: cfg,
		wg:  sync.WaitGroup{},
	}
}

func (d *DefaultService) CreateFolder(folderPath string) error {
	path := filepath.Join(d.cfg.DirPath, folderPath)
	return os.MkdirAll(path, os.ModePerm)
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []RequestFile, downloader Downloader) chan DownloadResult {
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
			go func(f RequestFile) {
				defer func() {
					<-sem
					wg.Done()
				}()
				d.processFile(ctx, folderPath, f, len(files), counter, result, downloader)
			}(file)
		}
		wg.Wait()
		close(result)
	}()

	return result
}

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file RequestFile, total int, counter *atomic.Int32, result chan DownloadResult, downloader Downloader) {
	filePath := filepath.Join(d.cfg.DirPath, folderPath, file.Name)
	counter.Inc()

	dst, err := prepareFilepath(filePath)
	if err != nil {
		result <- DownloadResult{
			Result: &ResponseFile{
				Name: file.Name,
			},
			Index: int(counter.Load()),
			Total: total,
			Err:   &ErrPrepareFilepath{Err: err},
		}
		return
	}

	err = downloader.DownloadFile(ctx, file.TGFileID, dst)
	if err != nil {
		if err := os.Remove(filePath); err != nil {
			result <- DownloadResult{
				Result: &ResponseFile{
					Name: file.Name,
				},
				Index: int(counter.Load()),
				Total: total,
				Err:   &ErrDownloadFailed{Err: err},
			}
		}
		result <- DownloadResult{
			Result: &ResponseFile{
				Name: file.Name,
			},
			Index: int(counter.Load()),
			Total: total,
			Err:   &ErrDownloadFailed{Err: err},
		}
	}

	checksum, err := calculateChecksum(filePath)
	if err != nil {
		result <- DownloadResult{
			Result: &ResponseFile{
				Name: file.Name,
			},
			Index: int(counter.Load()),
			Total: total,
			Err:   ErrCalculateChecksum,
		}
	}

	result <- DownloadResult{
		Result: &ResponseFile{
			Name:     file.Name,
			Checksum: checksum,
			TGFileID: file.TGFileID,
		},
		Index: int(counter.Load()),
		Total: total,
		Err:   nil,
	}
}

func (d *DefaultService) ReadFiles(folderPath string) (chan ReadResult, error) {
	dst := filepath.Join(d.cfg.DirPath, folderPath)
	entries, err := os.ReadDir(dst)
	if err != nil {
		return nil, &ErrReadDir{Err: err}
	}

	result := make(chan ReadResult)
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 5)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		for _, entry := range entries {
			wg.Add(1)
			sem <- struct{}{}
			go func(entry os.DirEntry) {
				defer func() {
					<-sem
					wg.Done()
				}()

				if entry.IsDir() {
					return
				}

				path := filepath.Join(dst, entry.Name())

				file, err := os.Open(path)
				if err != nil {
					result <- ReadResult{
						Name: entry.Name(),
						Err:  &ErrOpenFile{Err: err},
					}
				}

				checksum, err := calculateChecksum(path)
				if err != nil {
					result <- ReadResult{
						Name: entry.Name(),
						Err:  ErrCalculateChecksum,
					}
				}

				result <- ReadResult{
					Name:     entry.Name(),
					Body:     file,
					Checksum: checksum,
				}
			}(entry)
		}
		wg.Wait()
		close(result)
	}()

	return result, nil
}

func (d *DefaultService) DeleteFolder(folderPath string) error {
	folderPath = filepath.Join(d.cfg.DirPath, folderPath)
	return os.RemoveAll(folderPath)
}
