package file

import "print3d-order-bot/internal/pkg/model"

type DownloadResult struct {
	model.File
	Err error
}
