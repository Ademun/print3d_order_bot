package order

import (
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

func createFolderPath(clientName string, createdAt time.Time, orderID int) string {
	path := strings.Join([]string{clientName, createdAt.Format("2006-01-02"), strconv.Itoa(orderID)}, "_")
	return slug.Make(path)
}

func getFilenames(files []DBOrderFile) []string {
	filenames := make([]string, len(files))
	for i, filename := range files {
		filenames[i] = filename.FileName
	}
	return filenames
}
