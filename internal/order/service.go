package order

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/pkg/model"
	"strings"
	"time"

	"github.com/go-telegram/bot"
)

type Service interface {
	NewOrder(ctx context.Context, order model.Order, files []model.OrderFile) error
}

type DefaultService struct {
	repo Repo
}

func NewDefaultService(repo Repo) Service {
	return &DefaultService{repo: repo}
}

func (d *DefaultService) NewOrder(ctx context.Context, order model.Order, files []model.OrderFile) error {
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

	tx, err := d.repo.NewOrderOpenTx(ctx, dbOrder, files)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	bot.Bot{}.File

	// TODO: File Processing

	if err := d.repo.NewOrderCloseTX(tx); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
