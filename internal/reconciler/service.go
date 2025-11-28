package reconciler

import (
	"context"
	"log/slog"
	"os"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/model"
	"slices"
	"sync"
	"time"
)

type Service interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
	ReconcileOrder(ctx context.Context, orderID int)
}

type DefaultService struct {
	orderService order.Service
	fileService  file.Service
	wg           *sync.WaitGroup
}

func NewDefaultService(orderService order.Service, fileService file.Service) Service {
	return &DefaultService{
		orderService: orderService,
		fileService:  fileService,
		wg:           &sync.WaitGroup{},
	}
}

func (d *DefaultService) Start(ctx context.Context) {
	d.wg.Add(1)
	d.startReconciliationLoop(ctx)
}

func (d *DefaultService) Stop(ctx context.Context) error {
	stop := make(chan struct{})
	go func() {
		d.wg.Wait()
		stop <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stop:
		return nil
	}
}

func (d *DefaultService) startReconciliationLoop(ctx context.Context) {
	ticker := time.Tick(time.Hour)

	for {
		select {
		case <-ctx.Done():
			d.wg.Done()
			return
		case <-ticker:
			d.runGlobalReconciliation(ctx)
		}
	}
}

func (d *DefaultService) runGlobalReconciliation(ctx context.Context) {
	orders, err := d.orderService.GetActiveOrders(ctx)
	if err != nil {
		slog.Error(err.Error())
	}

	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 10)
	for _, ord := range orders {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.ReconcileOrder(ctx, ord.OrderID)
			<-sem
		}()
	}
	wg.Wait()
}

func (d *DefaultService) ReconcileOrder(ctx context.Context, orderID int) {
	order, err := d.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return
	}
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

	var newFiles []model.OrderFile
	for _, filename := range actualFilenames {
		if !slices.Contains(orderFilenames, filename) {
			newFiles = append(newFiles, model.OrderFile{
				FileName: filename,
			})
		}
	}

	var removedFilenames []string
	for _, filename := range orderFilenames {
		if !slices.ContainsFunc(newFiles, func(orderFile model.OrderFile) bool {
			return orderFile.FileName == filename
		}) {
			removedFilenames = append(removedFilenames, filename)
		}
	}

	if err := d.orderService.AddFilesToOrder(ctx, order.OrderID, newFiles); err != nil {
		slog.Error(err.Error())
	}

	if err := d.orderService.RemoveOrderFiles(ctx, order.OrderID, removedFilenames); err != nil {
		slog.Error(err.Error())
	}
}
