package mtproto

import (
	"context"
	"fmt"
	"io"
	"log"
	"print3d-order-bot/internal/mtproto/internal"
	"print3d-order-bot/pkg/config"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

type Client struct {
	api        *tg.Client
	uploader   *uploader.Uploader
	downloader *downloader.Downloader
	sender     *message.Sender
	cancel     context.CancelFunc
	done       chan struct{}
	ready      chan struct{}
}

func NewClient(ctx context.Context, cfg *config.MTProtoCfg) (*Client, error) {
	client := &Client{
		done:  make(chan struct{}),
		ready: make(chan struct{}),
	}

	clientCtx, cancel := context.WithCancel(ctx)
	client.cancel = cancel

	mtprotoClient := telegram.NewClient(cfg.AppID, cfg.AppHash, telegram.Options{})

	go func() {
		defer close(client.done)

		err := mtprotoClient.Run(clientCtx, func(ctx context.Context) error {
			if _, err := mtprotoClient.Auth().Bot(ctx, cfg.Token); err != nil {
				return fmt.Errorf("auth failed: %w", err)
			}

			api := tg.NewClient(mtprotoClient)
			u := uploader.NewUploader(api).WithPartSize(524288)
			d := downloader.NewDownloader().WithPartSize(524288)
			s := message.NewSender(api)

			client.api = api
			client.downloader = d
			client.uploader = u
			client.sender = s

			close(client.ready)

			<-ctx.Done()
			return ctx.Err()
		})

		if err != nil {
			log.Printf("MTProto client stopped with error: %v", err)
		}
	}()

	select {
	case <-client.ready:
		return client, nil
	case <-time.After(30 * time.Second):
		client.cancel()
		return nil, fmt.Errorf("client initialization timeout")
	case <-ctx.Done():
		client.cancel()
		return nil, ctx.Err()
	}
}

func (c *Client) UploadFile(ctx context.Context, filename string, file io.ReadCloser, userID int64) error {
	defer file.Close()
	upload, err := c.uploader.FromReader(ctx, filename, file)
	if err != nil {
		return err
	}

	document := message.UploadedDocument(upload).Filename(filename)

	peer := &tg.InputPeerUser{
		UserID:     userID,
		AccessHash: 0,
	}

	if _, err := c.sender.To(peer).Media(ctx, document); err != nil {
		return err
	}
	return nil
}

func (c *Client) DownloadFile(ctx context.Context, fileID string, dst io.Writer) error {
	fileInfo, err := internal.ParseFileID(fileID)
	if err != nil {
		return err
	}

	if fileInfo.ID == nil {
		return fmt.Errorf("file has no file id")
	}

	var location tg.InputFileLocationClass
	switch fileInfo.Type {
	case internal.IDPhoto:
		if fileInfo.PhotoInfo == nil {
			return fmt.Errorf("file has no photo info")
		}
		var thumbSize string
		switch source := fileInfo.PhotoInfo.PhotoSizeSource.(type) {
		case *internal.PhotoSizeSourceThumbnail:
			thumbSize = source.ThumbnailSize
		default:
			thumbSize = "y"
		}

		location = &tg.InputPhotoFileLocation{
			ID:            int64(*fileInfo.ID),
			AccessHash:    int64(fileInfo.AccessHash),
			FileReference: fileInfo.FileReference,
			ThumbSize:     thumbSize,
		}
	case internal.IDDocument, internal.IDVoice, internal.IDVideo, internal.IDAudio, internal.IDSticker:
		location = &tg.InputDocumentFileLocation{
			ID:            int64(*fileInfo.ID),
			AccessHash:    int64(fileInfo.AccessHash),
			FileReference: fileInfo.FileReference,
		}
	default:
		return fmt.Errorf("unsupported file type")
	}

	_, err = c.downloader.Download(c.api, location).Stream(ctx, dst)

	return err
}

func (c *Client) Close() error {
	c.cancel()
	<-c.done
	return nil
}
