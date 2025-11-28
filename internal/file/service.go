package file

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/pkg/model"
	"sync"

	"github.com/go-telegram/bot"
)

type Service interface {
	DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error
}

type DefaultService struct {
	client *http.Client
}

func NewDefaultService(client *http.Client) Service {
	return &DefaultService{client: client}
}

func (d *DefaultService) DownloadAndSave(ctx context.Context, folderPath string, files []model.OrderFile) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(files))
	// TODO: research optimal value and make it configurable
	sem := make(chan struct{}, 5)
	for _, file := range files {
		sem <- struct{}{}
		wg.Add(1)
		go d.processFile(ctx, folderPath, file, &wg)
	}
	wg.Wait()

	return nil
}

func (d *DefaultService) processFile(ctx context.Context, folderPath string, file model.OrderFile, wg *sync.WaitGroup) error {
	defer wg.Done()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if file.TGFileID == nil {
			// TODO: save
			return nil
		}
		fileUrl := getFileUrl(*file.TGFileID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileUrl, nil)
		if err != nil {
			// TODO: implement error
			panic(err)
		}
		resp, err := d.client.Do(req)
		if err != nil {
			// TODO: implement error
			panic(err)
		}
		defer resp.Body.Close()

		filePath := filepath.Join(folderPath, *file.TGFileID)
		out, err := os.Create(filePath)
		if err != nil {
			// TODO: implement error
			panic(err)
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			// TODO: implement error
			panic(err)
		}
	}
	return nil
}

func getFileUrl(tgFileID string) string {
	// TODO: implement
	return ""
}
