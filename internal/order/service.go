package order

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/pkg/model"
	"strings"
	"time"
)

type Service interface {
	NewOrder(ctx context.Context, order model.TGOrder, files []model.TGOrderFile) error
	AddFilesToOrder(ctx context.Context, orderID int, files []model.TGOrderFile) error
	GetOrderFilenames(ctx context.Context, orderID int) ([]string, error)
	GetActiveOrders(ctx context.Context) ([]model.Order, error)
	GetOrderByID(ctx context.Context, orderID int) (*model.Order, error)
	RemoveOrderFiles(ctx context.Context, orderID int, filenames []string) error
}

type DefaultService struct {
	repo        Repo
	fileService file.Service
}

func NewDefaultService(repo Repo, fileService file.Service) Service {
	return &DefaultService{
		repo:        repo,
		fileService: fileService,
	}
}

func (d *DefaultService) NewOrder(ctx context.Context, order model.TGOrder, files []model.TGOrderFile) error {
	createdAt := time.Now().Format("2006-01-02")
	dbOrder := DBOrder{
		ClientName: order.ClientName,
		CreatedAt:  createdAt,
		ClosedAt:   nil,
	}

	if strings.TrimSpace(order.Comments) != "" {
		files = append(files, model.TGOrderFile{
			FileName: "comments.txt",
			FileBody: &order.Comments,
			TGFileID: nil,
		})
	}

	folderPath, tx, err := d.repo.NewOrderOpenTx(ctx, dbOrder, files)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	if err := d.fileService.DownloadAndSave(ctx, folderPath, files); err != nil {
		if err := d.repo.NewOrderRollbackTX(tx); err != nil {
			slog.Error(err.Error())
			return err
		}
		slog.Error(err.Error())
		return err
	}

	if err := d.repo.NewOrderCloseTX(tx); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func (d *DefaultService) AddFilesToOrder(ctx context.Context, orderID int, files []model.TGOrderFile) error {
	if err := d.repo.NewOrderFiles(ctx, orderID, files); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func (d *DefaultService) GetOrderFilenames(ctx context.Context, orderID int) ([]string, error) {
	dbFiles, err := d.repo.GetOrderFiles(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	filenames := getFilenames(dbFiles)

	return filenames, nil
}

func (d *DefaultService) GetActiveOrders(ctx context.Context) ([]model.Order, error) {
	dbOrders, err := d.repo.GetOrders(ctx, true)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	orders := make([]model.Order, len(dbOrders))
	for i, dbOrder := range dbOrders {
		files, err := d.repo.GetOrderFiles(ctx, dbOrder.OrderID)
		if err != nil {
			slog.Error(err.Error())
			return nil, err
		}

		filenames := getFilenames(files)
		orders[i] = model.Order{
			OrderID:     dbOrder.OrderID,
			OrderStatus: dbOrder.OrderStatus,
			ClientName:  dbOrder.ClientName,
			CreatedAt:   dbOrder.CreatedAt,
			ClosedAt:    dbOrder.ClosedAt,
			FolderPath:  dbOrder.FolderPath,
			Filenames:   filenames,
		}
	}

	return orders, nil
}

func (d *DefaultService) GetOrderByID(ctx context.Context, orderID int) (*model.Order, error) {
	dbOrder, err := d.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	orderFiles, err := d.repo.GetOrderFiles(ctx, dbOrder.OrderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	filenames := getFilenames(orderFiles)

	order := &model.Order{
		OrderID:     dbOrder.OrderID,
		OrderStatus: dbOrder.OrderStatus,
		ClientName:  dbOrder.ClientName,
		CreatedAt:   dbOrder.CreatedAt,
		ClosedAt:    dbOrder.ClosedAt,
		FolderPath:  dbOrder.FolderPath,
		Filenames:   filenames,
	}

	return order, nil
}

func (d *DefaultService) RemoveOrderFiles(ctx context.Context, orderID int, filenames []string) error {
	if err := d.repo.DeleteOrderFiles(ctx, orderID, filenames); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
