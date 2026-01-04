package file

import (
	"context"
	"io"
)

type Downloader interface {
	DownloadFile(ctx context.Context, fileID string, dst io.Writer) error
}
