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

type BotApiDownloader struct {
	bot *bot.Bot
}

func NewBotApiDownloader(bot *bot.Bot) *BotApiDownloader {
	return &BotApiDownloader{
		bot: bot,
	}
}

func (d *BotApiDownloader) DownloadFile(ctx context.Context, fileID string, dst io.Writer) error {
	file, err := d.bot.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return err
	}

	link := d.bot.FileDownloadLink(file)
	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return err
	}

	return nil
}
