package model

type TGOrder struct {
	ClientName string
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
	ClosedAt    *string
	FolderPath  string
	Comments    *string
	Contacts    []string
	Links       []string
	Filenames   []string
}

type TGOrderFile struct {
	FileName string
	FileBody *string
	TGFileID *string
}
