package media

import (
	"fmt"
	"print3d-order-bot/internal/telegram/internal/model"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
)

func HasMedia(message *models.Message) bool {
	if message.Audio != nil {
		return true
	}
	if message.Photo != nil && len(message.Photo) > 0 {
		return true
	}

	if message.Document != nil {
		return true
	}

	if message.Video != nil {
		return true
	}

	if message.VideoNote != nil {
		return true
	}

	if message.Voice != nil {
		return true
	}

	if message.Entities != nil {
		result := false
		for _, entity := range message.Entities {
			if entity.Type == models.MessageEntityTypeURL {
				result = true
				break
			}
			if entity.Type == models.MessageEntityTypeEmail {
				result = true
				break
			}
			if entity.Type == models.MessageEntityTypePhoneNumber {
				result = true
				break
			}
		}
		return result
	}

	return false
}

func ExtractMedia(message *models.Message) []model.File {
	var result []model.File
	dateStr := time.Now().Format("2006-01-02")

	if message.Audio != nil {
		fileName := message.Audio.FileName
		if strings.TrimSpace(fileName) == "" {
			ext := getExtFromMIME(message.Audio.MimeType)
			fileName = fmt.Sprintf("audio_%s_%d%s", dateStr, message.ID, ext)
		}
		result = append(result, model.File{
			Name:     fileName,
			Size:     uint64(message.Audio.FileSize),
			TGFileID: message.Audio.FileID,
		})
	}

	if message.Photo != nil && len(message.Photo) > 0 {
		result = append(result, model.File{
			Name:     fmt.Sprintf("photo_%s_%d.jpg", dateStr, message.ID),
			Size:     uint64(message.Photo[len(message.Photo)-1].FileSize),
			TGFileID: message.Photo[len(message.Photo)-1].FileID,
		})
	}

	if message.Document != nil {
		fileName := message.Document.FileName
		if strings.TrimSpace(fileName) == "" {
			ext := getExtFromMIME(message.Document.MimeType)
			fileName = fmt.Sprintf("document_%s_%d%s", dateStr, message.ID, ext)
		}
		result = append(result, model.File{
			Name:     fileName,
			Size:     uint64(message.Document.FileSize),
			TGFileID: message.Document.FileID,
		})
	}

	if message.Video != nil {
		fileName := message.Video.FileName
		if strings.TrimSpace(fileName) == "" {
			ext := getExtFromMIME(message.Video.MimeType)
			fileName = fmt.Sprintf("video_%s_%d%s", dateStr, message.ID, ext)
		}
		result = append(result, model.File{
			Name:     fileName,
			Size:     uint64(message.Video.FileSize),
			TGFileID: message.Video.FileID,
		})
	}

	if message.VideoNote != nil {
		result = append(result, model.File{
			Name:     fmt.Sprintf("video_note_%s_%d.mp4", dateStr, message.ID),
			Size:     uint64(message.VideoNote.FileSize),
			TGFileID: message.VideoNote.FileID,
		})
	}

	if message.Voice != nil {
		ext := getExtFromMIME(message.Voice.MimeType)
		result = append(result, model.File{
			Name:     fmt.Sprintf("voice_%s_%d%s", dateStr, message.ID, ext),
			Size:     uint64(message.Voice.FileSize),
			TGFileID: message.Voice.FileID,
		})
	}

	return result
}

func ExtractResources(message *models.Message) ([]string, []string) {
	var contacts []string
	var links []string

	if message.Entities == nil {
		return contacts, links
	}

	for _, entity := range message.Entities {
		switch entity.Type {
		case models.MessageEntityTypeEmail, models.MessageEntityTypePhoneNumber:
			body := extractEntityText(message.Text, entity.Offset, entity.Length)
			contacts = append(contacts, body)
		case models.MessageEntityTypeTextLink:
			links = append(links, entity.URL)
		case models.MessageEntityTypeURL:
			body := extractEntityText(message.Text, entity.Offset, entity.Length)
			links = append(links, body)
		}
	}

	return contacts, links
}

func extractEntityText(text string, offset, length int) string {
	runes := []rune(text)
	utf16pos := 0
	utf16start := -1
	utf16end := -1

	for i, r := range runes {
		if utf16pos == offset {
			utf16start = i
		}

		if r > 0xFFFF {
			utf16pos += 2
		} else {
			utf16pos += 1
		}

		if utf16pos == offset+length {
			utf16end = i + 1
			break
		}
	}

	if utf16start == -1 || utf16end == -1 {
		return ""
	}

	return string(runes[utf16start:utf16end])
}

func getExtFromMIME(mimeType string) string {
	mimeMap := map[string]string{
		"audio/mpeg":      ".mp3",
		"audio/ogg":       ".ogg",
		"audio/mp4":       ".m4a",
		"video/mp4":       ".mp4",
		"video/quicktime": ".mov",
		"video/x-msvideo": ".avi",
		"video/webm":      ".webm",
		"application/pdf": ".pdf",
		"image/jpeg":      ".jpg",
		"image/png":       ".png",
		"image/gif":       ".gif",
		"image/webp":      ".webp",
	}

	if ext, ok := mimeMap[mimeType]; ok {
		return ext
	}

	parts := strings.Split(mimeType, "/")
	if len(parts) == 2 {
		return "." + parts[1]
	}

	return ".bin"
}
