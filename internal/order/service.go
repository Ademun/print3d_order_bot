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
	NewOrder(ctx context.Context, order model.TGOrder, files []model.OrderFile) error
	GetOrderFiles(ctx context.Context, orderID int) ([]model.OrderFile, error)
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

func (d *DefaultService) NewOrder(ctx context.Context, order model.TGOrder, files []model.OrderFile) error {
	createdAt := time.Now().Format("2006-01-02")
	dbOrder := DBOrder{
		ClientName: order.ClientName,
		CreatedAt:  createdAt,
		ClosedAt:   nil,
	}

	if strings.TrimSpace(order.Comments) != "" {
		files = append(files, model.OrderFile{
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

func (d *DefaultService) GetOrderFiles(ctx context.Context, orderID int) ([]model.OrderFile, error) {
	files, err := d.repo.GetOrderFiles(ctx, orderID)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return files, nil
}
