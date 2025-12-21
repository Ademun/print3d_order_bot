package order

import (
	"time"
)

type File struct {
	Name     string
	Checksum uint64
	TgFileID *string
}

type RequestNewOrder struct {
	ClientName string
	Cost       float32
	Comments   []string
	Contacts   []string
	Links      []string
	CreatedAt  time.Time
	FolderPath string
}

type RequestEditOrder struct {
	ClientName       *string
	Cost             *float32
	Comments         []string
	OverrideComments *bool
}

type Status string

const (
	StatusActive Status = "active"
	StatusClosed        = "closed"
)

type ResponseOrder struct {
	ID         int
	Status     Status
	ClientName string
	Cost       float32
	Comments   []string
	Contacts   []string
	Links      []string
	CreatedAt  time.Time
	ClosedAt   *time.Time
	FolderPath string
	Files      []File
}

type DBNewOrder struct {
	ID         int        `db:"id"`
	Status     Status     `db:"status"`
	ClientName string     `db:"client_name"`
	Cost       float32    `db:"cost"`
	Comments   []string   `db:"comments"`
	Contacts   []string   `db:"contacts"`
	Links      []string   `db:"links"`
	CreatedAt  time.Time  `db:"created_at"`
	ClosedAt   *time.Time `db:"closed_at"`
	FolderPath string     `db:"folder_path"`
}

type DBEditOrder struct {
	ID         int      `db:"id"`
	ClientName *string  `db:"client_name"`
	Cost       *float32 `db:"cost"`
	Comments   []string
}

type DBFile struct {
	Name     string  `db:"name"`
	Checksum uint64  `db:"checksum"`
	TgFileID *string `db:"tg_file_id"`
	OrderID  int     `db:"order_id"`
}
