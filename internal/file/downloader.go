package file

import (
	"context"
	"io"
	"print3d-order-bot/internal/mtproto"
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
