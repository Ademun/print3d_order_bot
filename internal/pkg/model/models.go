package model

import (
	"net/url"
)

type TGOrder struct {
	ClientName string
	Comments   string
}

type OrderStatus int

const (
	StatusActive OrderStatus = iota
	StatusClosed
)

type Order struct {
	OrderID     int
	OrderStatus OrderStatus
	ClientName  string
	CreatedAt   string
	ClosedAt    string
	FolderPath  string
	Comments    string
	Resources   []url.URL
	Filenames   []string
}

type TGOrderFile struct {
	FileName string
	FileBody *string
	TGFileID *string
}
