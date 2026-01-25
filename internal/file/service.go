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
	SetDownloaders(botApiDownloader, mtprotoDownloader Downloader)
	CreateFolder(folderPath string) error
	DownloadAndSave(ctx context.Context, folderPath string, files []RequestFile) chan DownloadResult
	ReadFiles(folderPath string) (chan ReadResult, error)
	DeleteFolder(folderPath string) error
}

type DefaultService struct {
	botApiDownloader  Downloader
	mtprotoDownloader Downloader
	cfg               *config.FileServiceCfg
	wg                sync.WaitGroup
}

func NewDefaultService(cfg *config.FileServiceCfg) Service {
	return &DefaultService{
		cfg: cfg,
		wg:  sync.WaitGroup{},
	}
}

func (d *DefaultService) SetDownloaders(botApiDownloader, mtprotoDownloader Downloader) {
	d.botApiDownloader = botApiDownloader
	d.mtprotoDownloader = mtprotoDownloader
}

func (d *DefaultService) CreateFolder(folderPath string) error {
	path := filepath.Join(d.cfg.DirPath, folderPath)
	return os.MkdirAll(path, os.ModePerm)
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []RequestFile) chan DownloadResult {
	wg := sync.WaitGroup{}
	counter := atomic.NewInt32(0)
	result := make(chan DownloadResult)
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
				d.processFile(ctx, folderPath, f, len(files), counter, result)
			}(file)
		}
		wg.Wait()
		close(result)
	}()

	return result
}

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file RequestFile, total int, counter *atomic.Int32, result chan DownloadResult) {
	filePath := filepath.Join(d.cfg.DirPath, folderPath, file.Name)
	currentIndex := int(counter.Inc())

	dst, err := prepareFilepath(filePath)
	if err != nil {
		result <- DownloadResult{
			Result: &ResponseFile{
				Name: file.Name,
			},
			Index: currentIndex,
			Total: total,
			Err:   &ErrPrepareFilepath{Err: err},
		}
		return
	}

	var downloadErr error
	if file.Size <= 19*1024*1024 {
		downloadErr = d.botApiDownloader.DownloadFile(ctx, file.TGFileID, dst)
	} else {
		downloadErr = d.mtprotoDownloader.DownloadFile(ctx, file.TGFileID, dst)
	}

	if downloadErr != nil {
		if err := os.Remove(filePath); err != nil {
			result <- DownloadResult{
				Result: &ResponseFile{
					Name: file.Name,
				},
				Index: currentIndex,
				Total: total,
				Err:   &ErrDownloadFailed{Err: err},
			}
		}
		result <- DownloadResult{
			Result: &ResponseFile{
				Name: file.Name,
			},
			Index: currentIndex,
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
			Index: currentIndex,
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
		Index: currentIndex,
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

				fileInfo, err := entry.Info()
				if err != nil {
					result <- ReadResult{
						Name: entry.Name(),
						Err:  &ErrOpenFile{Err: err},
					}
				}

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
					Size:     uint64(fileInfo.Size()),
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
