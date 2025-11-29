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
	GetOrderFiles(ctx context.Context, orderID int) ([]model.TGOrderFile, error)
	GetActiveOrders(ctx context.Context) ([]DBOrder, error)
	GetOrderByID(ctx context.Context, orderID int) (*DBOrder, error)
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

func (d *DefaultService) GetOrderFiles(ctx context.Context, orderID int) ([]model.TGOrderFile, error) {
	files, err := d.repo.GetOrderFiles(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return files, nil
}

func (d *DefaultService) GetActiveOrders(ctx context.Context) ([]DBOrder, error) {
	orders, err := d.repo.GetOrders(ctx, true)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return orders, nil
}

func (d *DefaultService) GetOrderByID(ctx context.Context, orderID int) (*DBOrder, error) {
	orders, err := d.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return orders, nil
}

func (d *DefaultService) RemoveOrderFiles(ctx context.Context, orderID int, filenames []string) error {
	if err := d.repo.DeleteOrderFiles(ctx, orderID, filenames); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
