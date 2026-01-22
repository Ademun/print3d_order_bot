package telegram

import (
	"strconv"
	"strings"
	"time"
)

func CreateFolderPath(createdAt time.Time, clientName string, comments string, printType string) string {
	return strings.Join([]string{createdAt.Format("02.01.2006"), clientName, comments, printType, strconv.FormatInt(createdAt.Unix(), 10)}, " ")
}
