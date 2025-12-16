package file

import (
	"io"
	"print3d-order-bot/internal/pkg/model"
)

type DownloadResult struct {
	Result *model.File
	Index  int
	Total  int
	Err    error
}

type ReadResult struct {
	Name     string
	Body     io.ReadCloser
	Checksum uint64
	Err      error
}
