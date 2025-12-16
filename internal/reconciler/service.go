package reconciler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/config"
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
	cfg          *config.FileServiceCfg
	wg           *sync.WaitGroup
}

func NewDefaultService(orderService order.Service, fileService file.Service, cfg *config.FileServiceCfg) Service {
	return &DefaultService{
		orderService: orderService,
		fileService:  fileService,
		cfg:          cfg,
		wg:           &sync.WaitGroup{},
	}
}

func (d *DefaultService) Start(ctx context.Context) {
	d.startReconciliationLoop(ctx)
	slog.Info("Started reconciler service")
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

	d.wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				d.wg.Done()
				return
			case <-ticker:
				d.runGlobalReconciliation(ctx)
			}
		}
	}()
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
		go func(o model.Order) {
			defer wg.Done()
			d.ReconcileOrder(ctx, o.OrderID)
			<-sem
		}(ord)
	}
	wg.Wait()

	validFolders := make([]string, len(orders))
	for i, ord := range orders {
		validFolders[i] = ord.FolderPath
	}

	entries, err := os.ReadDir(d.cfg.DirPath)
	if err != nil {
		slog.Error(err.Error())
	}

	for _, dirEntry := range entries {
		if !slices.Contains(validFolders, dirEntry.Name()) {
			path := filepath.Join(d.cfg.DirPath, dirEntry.Name())
			if err := os.RemoveAll(path); err != nil {
				slog.Error(err.Error())
			}
		}
	}
}

func (d *DefaultService) ReconcileOrder(ctx context.Context, orderID int) {
	order, err := d.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return
	}
	orderFilenames, err := d.orderService.GetOrderFilenames(ctx, order.OrderID)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	path := filepath.Join(d.cfg.DirPath, order.FolderPath)
	dirFiles, err := os.ReadDir(path)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	actualFilenames := make([]string, len(dirFiles))
	for i, file := range dirFiles {
		actualFilenames[i] = file.Name()
	}

	var newFiles []model.File
	for _, filename := range actualFilenames {
		if !slices.Contains(orderFilenames, filename) {
			newFiles = append(newFiles, model.File{
				Name: filename,
			})
		}
	}

	var removedFilenames []string
	for _, filename := range orderFilenames {
		if !slices.Contains(actualFilenames, filename) {
			removedFilenames = append(removedFilenames, filename)
		}
	}

	if len(newFiles) > 0 {
		if err := d.orderService.AddFilesToOrder(ctx, order.OrderID, newFiles); err != nil {
			slog.Error(err.Error())
		}
	}

	if len(removedFilenames) > 0 {
		if err := d.orderService.RemoveOrderFiles(ctx, order.OrderID, removedFilenames); err != nil {
			slog.Error(err.Error())
		}
	}
}
