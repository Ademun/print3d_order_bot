package order

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type Service interface {
	NewOrder(ctx context.Context, order TGOrder, files []TGOrderFile) error
}

type DefaultService struct {
	repo Repo
}

func NewDefaultService(repo Repo) Service {
	return &DefaultService{repo: repo}
}

func (d *DefaultService) NewOrder(ctx context.Context, order TGOrder, files []TGOrderFile) error {
	createdAt := time.Now().Format("2006-01-02")
	dbOrder := DBOrder{
		ClientName: order.ClientName,
		CreatedAt:  createdAt,
		ClosedAt:   nil,
	}

	if strings.TrimSpace(order.Comments) != "" {
		files = append(files, TGOrderFile{
			FileName: "comments.txt",
			FileID:   nil,
		})
	}

	tx, err := d.repo.NewOrderOpenTx(ctx, dbOrder, files)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// TODO: File Processing

	if err := d.repo.NewOrderCloseTX(tx); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
