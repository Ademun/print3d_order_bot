package file

import (
	"io"
)

type RequestFile struct {
	Name     string
	Size     uint64
	TGFileID string
}

type ResponseFile struct {
	Name     string
	TGFileID string
}

type DownloadResult struct {
	Result *ResponseFile
	Index  int
	Total  int
	Err    error
}

type ReadResult struct {
	Name string
	Body io.ReadCloser
	Size uint64
	Err  error
}
