package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"print3d-order-bot/internal/mtproto"

	"github.com/go-telegram/bot"
)

type Downloader interface {
	DownloadFile(ctx context.Context, fileID string, dst io.Writer) error
}

type MTProtoDownloader struct {
	client *mtproto.Client
}

func NewMTProtoDownloader(client *mtproto.Client) *MTProtoDownloader {
	return &MTProtoDownloader{
		client: client,
	}
}

func (d *MTProtoDownloader) DownloadFile(ctx context.Context, fileID string, dst io.Writer) error {
	return d.client.DownloadFile(ctx, fileID, dst)
}

type BotAPIDownloader struct {
	api *bot.Bot
}

func NewBotAPIDownloader(api *bot.Bot) *BotAPIDownloader {
	return &BotAPIDownloader{
		api: api,
	}
}

func (d *BotAPIDownloader) DownloadFile(ctx context.Context, fileID string, dst io.Writer) error {
	fileInfo, err := d.api.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})

	if fileInfo.FileSize > 20*1024*1024 {
		return fmt.Errorf("file is too large")
	}

	if err != nil {
		return err
	}

	link := d.api.FileDownloadLink(fileInfo)

	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}
