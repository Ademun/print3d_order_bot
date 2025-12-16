package model

import (
	"io"
	"time"
)

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

type File struct {
	Name     string
	Checksum uint64
	TGFileID *string
}

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
	Files       []File
}

type FileResult struct {
	Filename string
	File     io.ReadCloser
	Checksum uint64
	Err      error
}
