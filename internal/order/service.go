package order

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/pkg/model"
	"time"
)

type Service interface {
	NewOrder(ctx context.Context, order RequestOrder, files []File) error
	AddFilesToOrder(ctx context.Context, orderID int, files []File) error
	GetOrderFilenames(ctx context.Context, orderID int) ([]string, error)
	GetActiveOrdersIDs(ctx context.Context) ([]int, error)
	GetOrderByID(ctx context.Context, orderID int) (*ResponseOrder, error)
	CloseOrder(ctx context.Context, orderID int) error
	RestoreOrder(ctx context.Context, orderID int) error
	RemoveOrderFiles(ctx context.Context, orderID int, filenames []string) error
	UpdateOrderFiles(ctx context.Context, orderID int, files []File) error
}

type DefaultService struct {
	repo        Repo
	fileService file.Service
}

func NewDefaultService(repo Repo) Service {
	return &DefaultService{
		repo: repo,
	}
}

func (d *DefaultService) NewOrder(ctx context.Context, order RequestOrder, files []File) error {
	dbOrder := DBOrder{
		Status:     model.StatusActive,
		ClientName: order.ClientName,
		Cost:       order.Cost,
		Comments:   order.Comments,
		Contacts:   order.Contacts,
		Links:      order.Links,
		CreatedAt:  order.CreatedAt,
	}

	dbFiles := make([]DBFile, len(files))
	for i, file := range files {
		dbFiles[i] = DBFile{
			Name:     file.Name,
			Checksum: file.Checksum,
			TgFileID: file.TgFileID,
		}
	}

	if err := d.repo.NewOrder(ctx, dbOrder, dbFiles); err != nil {
		slog.Error("Failed to create new order", "error", err)
		return err
	}

	return nil
}

func (d *DefaultService) AddFilesToOrder(ctx context.Context, orderID int, files []File) error {
	dbFiles := make([]DBFile, len(files))
	for i, file := range files {
		dbFiles[i] = DBFile{
			Name:     file.Name,
			Checksum: file.Checksum,
			TgFileID: file.TgFileID,
			OrderID:  orderID,
		}
	}

	if err := d.repo.AddFilesToOrder(ctx, orderID, dbFiles); err != nil {
		slog.Error("Failed to add files to order", "error", err, "orderID", orderID)
		return err
	}

	return nil
}

func (d *DefaultService) GetOrderFilenames(ctx context.Context, orderID int) ([]string, error) {
	filenames, err := d.repo.GetOrderFilenames(ctx, orderID)
	if err != nil {
		slog.Error("Failed to retrieve order filenames", "error", err, "orderID", orderID)
		return nil, err
	}

	return filenames, nil
}

func (d *DefaultService) GetActiveOrdersIDs(ctx context.Context) ([]int, error) {
	ids, err := d.repo.GetOrdersIDs(ctx, true)
	if err != nil {
		slog.Error("Error retrieving active orders IDs", "error", err)
	}
	return ids, err
}

func (d *DefaultService) GetOrderByID(ctx context.Context, orderID int) (*ResponseOrder, error) {
	dbOrder, err := d.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error("Error retrieving order", "error", err, "orderID", orderID)
		return nil, err
	}

	dbFiles, err := d.repo.GetOrderFiles(ctx, orderID)
	if err != nil {
		slog.Error("Error retrieving order files", "error", err, "orderID", orderID)
		return nil, err
	}

	files := make([]File, len(dbFiles))
	for i, file := range dbFiles {
		files[i] = File{
			Name:     file.Name,
			Checksum: file.Checksum,
			TgFileID: file.TgFileID,
		}
	}

	order := &ResponseOrder{
		ID:         dbOrder.ID,
		Status:     dbOrder.Status,
		ClientName: dbOrder.ClientName,
		Cost:       dbOrder.Cost,
		Comments:   dbOrder.Comments,
		Contacts:   dbOrder.Contacts,
		Links:      dbOrder.Links,
		CreatedAt:  dbOrder.CreatedAt,
		ClosedAt:   dbOrder.ClosedAt,
		FolderPath: dbOrder.FolderPath,
		Files:      files,
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

func (d *DefaultService) UpdateOrderFiles(ctx context.Context, orderID int, files []File) error {
	dbFiles := make([]DBFile, len(files))
	for i, file := range files {
		dbFiles[i] = DBFile{
			Name:     file.Name,
			Checksum: file.Checksum,
			TgFileID: file.TgFileID,
		}
	}

	if err := d.repo.UpdateOrderFiles(ctx, orderID, dbFiles); err != nil {
		slog.Error("Failed to update order files", "error", err, "orderID", orderID)
		return err
	}

	return nil
}
