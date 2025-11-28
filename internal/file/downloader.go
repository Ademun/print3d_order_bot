package file

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-telegram/bot"
)

type Downloader interface {
	DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error)
}

type TelegramDownloader struct {
	api    *bot.Bot
	client *http.Client
}

func NewTelegramDownloader(api *bot.Bot, client *http.Client) Downloader {
	return &TelegramDownloader{api: api, client: client}
}

func (d *TelegramDownloader) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	file, err := d.api.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("error geting file %s: %w", fileID, err)
	}

	link := d.api.FileDownloadLink(file)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for file %s: %w", fileID, err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading file %s: %w", fileID, err)
	}

	if resp.StatusCode != http.StatusOK {
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("error closing response body for file %s: %w", fileID, err)
		}
		return nil, fmt.Errorf("error downloading file %s: unexpected status code %d", fileID, resp.StatusCode)
	}

	return resp.Body, nil
}
