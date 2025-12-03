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
	AddFilesToOrder(orderID int, files []model.TGOrderFile) error
	GetOrderFilenames(orderID int) ([]string, error)
	GetActiveOrders() ([]model.Order, error)
	GetActiveOrdersIDs() ([]int, error)
	GetOrderByID(orderID int) (*model.Order, error)
	CloseOrder(orderID int) error
	RestoreOrder(orderID int) error
	RemoveOrderFiles(orderID int, filenames []string) error
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
	createdAt := time.Now()
	dbOrder := DBOrder{
		ClientName: order.ClientName,
		Comments:   order.Comments,
		Contacts:   order.Contacts,
		Links:      order.Links,
		CreatedAt:  createdAt,
	}

	folderPath, tx, err := d.repo.NewOrderOpenTx(dbOrder, files)
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

func (d *DefaultService) AddFilesToOrder(orderID int, files []model.TGOrderFile) error {
	if err := d.repo.NewOrderFiles(orderID, files); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func (d *DefaultService) GetOrderFilenames(orderID int) ([]string, error) {
	dbFiles, err := d.repo.GetOrderFiles(orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	filenames := getFilenames(dbFiles)

	return filenames, nil
}

func (d *DefaultService) GetActiveOrders() ([]model.Order, error) {
	dbOrders, err := d.repo.GetOrders(true)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	orders := make([]model.Order, len(dbOrders))
	for i, dbOrder := range dbOrders {
		files, err := d.repo.GetOrderFiles(dbOrder.OrderID)
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

func (d *DefaultService) GetActiveOrdersIDs() ([]int, error) {
	ids, err := d.repo.GetOrdersIDs(true)
	if err != nil {
		slog.Error(err.Error())
	}
	return ids, err
}

func (d *DefaultService) GetOrderByID(orderID int) (*model.Order, error) {
	dbOrder, err := d.repo.GetOrderByID(orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	orderFiles, err := d.repo.GetOrderFiles(dbOrder.OrderID)
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

func (d *DefaultService) CloseOrder(orderID int) error {
	if err := d.repo.UpdateOrderStatus(orderID, model.StatusClosed); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func (d *DefaultService) RestoreOrder(orderID int) error {
	if err := d.repo.UpdateOrderStatus(orderID, model.StatusActive); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}

func (d *DefaultService) RemoveOrderFiles(orderID int, filenames []string) error {
	if err := d.repo.DeleteOrderFiles(orderID, filenames); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
