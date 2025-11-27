package order

import (
	"strconv"
	"strings"

	"github.com/gosimple/slug"
)

func createFolderPath(clientName, createdAt string, orderID int) string {
	path := strings.Join([]string{clientName, createdAt, strconv.Itoa(orderID)}, "_")
	return slug.Make(path)
}
