package reconciler

import (
	"context"
	"log/slog"
	"os"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/order"
	"slices"
)

type Service interface {
	Start(ctx context.Context) error
}

type DefaultService struct {
	orderService order.Service
	fileService  file.Service
}

func NewDefaultService(orderService order.Service, fileService file.Service) Service {
	return &DefaultService{
		orderService: orderService,
		fileService:  fileService,
	}
}

func (d *DefaultService) Start(ctx context.Context) error {

}

func (d *DefaultService) ReconcileOrder(ctx context.Context, order order.DBOrder) {
	orderFiles, err := d.orderService.GetOrderFiles(ctx, order.OrderID)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	orderFilenames := make([]string, len(orderFiles))
	for i, file := range orderFiles {
		orderFilenames[i] = file.FileName
	}

	dirFiles, err := os.ReadDir(order.FolderPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	actualFilenames := make([]string, len(dirFiles))
	for i, file := range dirFiles {
		actualFilenames[i] = file.Name()
	}

	var newFilenames []string
	for _, file := range actualFilenames {
		if !slices.Contains(orderFilenames, file) {
			newFilenames = append(newFilenames, file)
		}
	}

	var removedFilenames []string
	for _, file := range orderFilenames {
		if !slices.Contains(newFilenames, file) {
			removedFilenames = append(removedFilenames, file)
		}
	}

}
