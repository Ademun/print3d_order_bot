package order

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/pkg/model"
	"time"
)

type Service interface {
	NewOrder(ctx context.Context, order model.TGOrder, files []model.TGOrderFile) error
	AddFilesToOrder(ctx context.Context, orderID int, files []model.TGOrderFile) error
	GetOrderFilenames(ctx context.Context, orderID int) ([]string, error)
	GetActiveOrders(ctx context.Context) ([]model.Order, error)
	GetActiveOrdersIDs(ctx context.Context) ([]int, error)
	GetOrderByID(ctx context.Context, orderID int) (*model.Order, error)
	CloseOrder(ctx context.Context, orderID int) error
	RestoreOrder(ctx context.Context, orderID int) error
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
	dbOrder := DBOrder{
		OrderStatus: model.StatusActive,
		ClientName:  order.ClientName,
		Cost:        order.Cost,
		Comments:    order.Comments,
		Contacts:    order.Contacts,
		Links:       order.Links,
		CreatedAt:   time.Now(),
	}

	folderPath, tx, err := d.repo.NewOrderOpenTx(ctx, dbOrder, files)
	if err != nil {
		slog.Error("Error creating new order", "error", err)
		return err
	}

	if err := d.fileService.CreateFolder(folderPath); err != nil {
		slog.Error("Error creating new order", "error", err)
		return err
	}

	if err := d.fileService.DownloadAndSave(ctx, folderPath, files); err != nil {
		if err := d.repo.NewOrderRollbackTX(ctx, tx); err != nil {
			slog.Error(err.Error())
			return err
		}
		slog.Error("Error creating new order", "error", err)
		return err
	}

	if err := d.repo.NewOrderCloseTX(ctx, tx); err != nil {
		slog.Error("Error creating new order", "error", err)
		return err
	}
	return nil
}

func (d *DefaultService) AddFilesToOrder(ctx context.Context, orderID int, files []model.TGOrderFile) error {
	if err := d.repo.NewOrderFiles(ctx, orderID, files); err != nil {
		slog.Error("Error adding files to order", "error", err, "orderID", orderID)
		return err
	}
	return nil
}

func (d *DefaultService) GetOrderFilenames(ctx context.Context, orderID int) ([]string, error) {
	dbFiles, err := d.repo.GetOrderFiles(ctx, orderID)
	if err != nil {
		slog.Error("Error retrieving order filenames", "error", err, "orderID", orderID)
		return nil, err
	}

	filenames := getFilenames(dbFiles)

	return filenames, nil
}

func (d *DefaultService) GetActiveOrders(ctx context.Context) ([]model.Order, error) {
	dbOrders, err := d.repo.GetOrders(ctx, true)
	if err != nil {
		slog.Error("Error retrieving active orders", "error", err)
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
			Cost:        dbOrder.Cost,
			CreatedAt:   dbOrder.CreatedAt,
			ClosedAt:    dbOrder.ClosedAt,
			FolderPath:  dbOrder.FolderPath,
			Filenames:   filenames,
		}
	}

	return orders, nil
}

func (d *DefaultService) GetActiveOrdersIDs(ctx context.Context) ([]int, error) {
	ids, err := d.repo.GetOrdersIDs(ctx, true)
	if err != nil {
		slog.Error("Error retrieving active orders IDs", "error", err)
	}
	return ids, err
}

func (d *DefaultService) GetOrderByID(ctx context.Context, orderID int) (*model.Order, error) {
	dbOrder, err := d.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error("Error retrieving order", "error", err, "orderID", orderID)
		return nil, err
	}

	orderFiles, err := d.repo.GetOrderFiles(ctx, dbOrder.OrderID)
	if err != nil {
		slog.Error("Error retrieving order", "error", err, "orderID", orderID)
		return nil, err
	}

	filenames := getFilenames(orderFiles)

	order := &model.Order{
		OrderID:     dbOrder.OrderID,
		OrderStatus: dbOrder.OrderStatus,
		ClientName:  dbOrder.ClientName,
		Cost:        dbOrder.Cost,
		Comments:    dbOrder.Comments,
		Contacts:    dbOrder.Contacts,
		Links:       dbOrder.Links,
		CreatedAt:   dbOrder.CreatedAt,
		ClosedAt:    dbOrder.ClosedAt,
		FolderPath:  dbOrder.FolderPath,
		Filenames:   filenames,
	}

	return order, nil
}

func (d *DefaultService) CloseOrder(ctx context.Context, orderID int) error {
	if err := d.repo.UpdateOrderStatus(ctx, orderID, model.StatusClosed); err != nil {
		slog.Error("Error closing order", "error", err, "orderID", orderID)
		return err
	}
	return nil
}

func (d *DefaultService) RestoreOrder(ctx context.Context, orderID int) error {
	order, err := d.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error("Error restoring order", "error", err, "orderID", orderID)
		return err
	}
	if order.ClosedAt != nil && time.Since(order.ClosedAt.UTC()).Hours() >= 24 {
		return ErrRestorationPeriodExpired
	}
	if err := d.repo.UpdateOrderStatus(ctx, orderID, model.StatusActive); err != nil {
		slog.Error("Error restoring order", "error", err, "orderID", orderID)
		return err
	}
	return nil
}

func (d *DefaultService) RemoveOrderFiles(ctx context.Context, orderID int, filenames []string) error {
	if err := d.repo.DeleteOrderFiles(ctx, orderID, filenames); err != nil {
		slog.Error("Error removing order files", "error", err, "orderID", orderID)
		return err
	}
	return nil
}
