package model

type TGOrder struct {
	ClientName string
	Comments   string
}

type Order struct {
}

type OrderFile struct {
	FileName string
	FileBody *string
	TGFileID *string
}
