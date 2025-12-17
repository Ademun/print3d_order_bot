package telegram

import (
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

func CreateFolderPath(clientName string, createdAt time.Time, messageID int) string {
	path := strings.Join([]string{clientName, createdAt.Format("2006-01-02"), strconv.Itoa(messageID)}, "_")
	return slug.Make(path)
}
