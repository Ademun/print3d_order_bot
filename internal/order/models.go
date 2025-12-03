package order

import (
	"print3d-order-bot/internal/pkg/model"
	"time"
)

type DBOrder struct {
	OrderID     int               `db:"order_id"`
	OrderStatus model.OrderStatus `db:"order_status"`
	ClientName  string            `db:"client_name"`
	Comments    []string          `db:"comments"`
	Contacts    []string          `db:"contacts"`
	Links       []string          `db:"links"`
	CreatedAt   time.Time         `db:"created_at"`
	ClosedAt    *time.Time        `db:"closed_at"`
	FolderPath  string            `db:"folder_path"`
}

type DBOrderFile struct {
	FileName string  `db:"file_name"`
	TgFileID *string `db:"tg_file_id"`
	OrderID  int     `db:"order_id"`
}
