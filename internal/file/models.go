package file

import "print3d-order-bot/internal/pkg/model"

type DownloadResult struct {
	Result *model.File
	Index  int
	Total  int
	Err    error
}
