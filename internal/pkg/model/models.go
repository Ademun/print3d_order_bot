package model

import "time"

type TGOrder struct {
	ClientName string
	Cost       float32
	Comments   []string
	Contacts   []string
	Links      []string
}

type OrderStatus string

const (
	StatusActive OrderStatus = "active"
	StatusClosed             = "closed"
)

type Order struct {
	OrderID     int
	OrderStatus OrderStatus
	CreatedAt   time.Time
	ClientName  string
	Cost        float32
	Comments    []string
	Contacts    []string
	Links       []string
	ClosedAt    *time.Time
	FolderPath  string
	Filenames   []string
}

type TGOrderFile struct {
	FileName string
	FileBody *string
	TGFileID *string
}
