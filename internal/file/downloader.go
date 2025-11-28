package file

import "io"

type Downloader interface {
	DownloadFile(fileID string) (io.ReadCloser, error)
}
