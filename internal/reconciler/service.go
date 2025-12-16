package reconciler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	fileSvc "print3d-order-bot/internal/file"
	orderSvc "print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/config"
	"sync"
	"time"
)

type Service interface {
	Start(ctx context.Context)
	Stop(ctx context.Context) error
	ReconcileOrder(ctx context.Context, orderID int)
}

type DefaultService struct {
	orderService orderSvc.Service
	fileService  fileSvc.Service
	cfg          *config.FileServiceCfg
	wg           *sync.WaitGroup
}

func NewDefaultService(orderService orderSvc.Service, fileService fileSvc.Service, cfg *config.FileServiceCfg) Service {
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
	orderIDs, err := d.orderService.GetActiveOrdersIDs(ctx)
	if err != nil {
		slog.Error(err.Error())
	}

	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 10)
	for _, id := range orderIDs {
		sem <- struct{}{}
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			d.ReconcileOrder(ctx, id)
			<-sem
		}(id)
	}
	wg.Wait()

	validFolders, err := d.orderService.GetActiveOrdersFolders(ctx)
	if err != nil {
		slog.Error(err.Error())
	}

	validFoldersMap := make(map[string]struct{})
	for _, folder := range validFolders {
		validFoldersMap[folder] = struct{}{}
	}

	entries, err := os.ReadDir(d.cfg.DirPath)
	if err != nil {
		slog.Error(err.Error())
	}

	for _, dirEntry := range entries {
		if !dirEntry.IsDir() {
			continue
		}

		if _, ok := validFoldersMap[dirEntry.Name()]; !ok {
			path := filepath.Join(d.cfg.DirPath, dirEntry.Name())
			if err := d.fileService.DeleteFolder(path); err != nil {
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

	orderFilesMap := make(map[string]orderSvc.File)
	for _, file := range order.Files {
		orderFilesMap[file.Name] = file
	}

	files, err := d.fileService.ReadFiles(order.FolderPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	var removedFiles []string
	var newFiles []orderSvc.File
	var updatedFiles []orderSvc.File

	filesMap := make(map[string]fileSvc.ReadResult)

	for file := range files {
		if file.Err != nil {
			slog.Error(file.Err.Error())
			continue
		}

		filesMap[file.Name] = file

		orderFile, ok := orderFilesMap[file.Name]
		if !ok {
			newFiles = append(newFiles, orderSvc.File{
				Name:     file.Name,
				Checksum: file.Checksum,
			})
			continue
		}

		if file.Checksum != orderFile.Checksum {
			updatedFiles = append(updatedFiles, orderSvc.File{
				Name:     file.Name,
				Checksum: file.Checksum,
			})
			delete(orderFilesMap, file.Name)
			continue
		}
	}

	for name, _ := range orderFilesMap {
		if _, ok := filesMap[name]; !ok {
			removedFiles = append(removedFiles, name)
		}
	}

	if err := d.orderService.RemoveOrderFiles(ctx, orderID, removedFiles); err != nil {
		slog.Error(err.Error())
	}

	if err := d.orderService.AddFilesToOrder(ctx, orderID, newFiles); err != nil {
		slog.Error(err.Error())
	}

	if err := d.orderService.UpdateOrderFiles(ctx, orderID, updatedFiles); err != nil {
		slog.Error(err.Error())
	}
}
