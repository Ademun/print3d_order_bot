package model

type Order struct {
	ClientName string
	Comments   string
}

type OrderFile struct {
	FileName string
	FileBody *string
	TGFileID *string
}
