package file

import (
	"context"
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
		// TODO: implement error
		panic(err)
	}

	link := d.api.FileDownloadLink(file)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		// TODO: implement error
		panic(err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// TODO: implement error
		panic(err)
	}

	return resp.Body, nil
}
