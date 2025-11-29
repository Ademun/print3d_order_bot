package order

import (
	"print3d-order-bot/internal/pkg/model"
)

type DBOrder struct {
	OrderID     int               `db:"order_id"`
	OrderStatus model.OrderStatus `db:"order_status"`
	ClientName  string            `db:"client_name"`
	CreatedAt   string            `db:"created_at"`
	ClosedAt    *string           `db:"closed_at"`
	FolderPath  string            `db:"folder_path"`
}

type DBOrderFile struct {
	FileName string  `db:"file_name"`
	TgFileID *string `db:"tg_file_id"`
	OrderID  int     `db:"order_id"`
}
