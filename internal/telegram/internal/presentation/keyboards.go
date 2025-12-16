package presentation

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

func OrderTypeKbd() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "Новый", CallbackData: "new_order"}},
			{{Text: "Старый", CallbackData: "old_order"}},
		},
	}
}

func SkipKbd() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "⏩ Пропустить", CallbackData: "skip"}},
		},
	}
}

func YesNoKbd() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{{Text: "✔️ Да", CallbackData: "yes"}},
			{{Text: "❌ Нет", CallbackData: "no"}},
		},
	}
}

type OrderSliderAction int

const (
	OrderSliderClose OrderSliderAction = iota
	OrderSliderRestore
)

func OrderSliderMgmtKbd(total, currentIdx int, action OrderSliderAction) *models.InlineKeyboardMarkup {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{},
	}
	var sliderRow []models.InlineKeyboardButton
	if currentIdx > 0 {
		sliderRow = append(sliderRow, models.InlineKeyboardButton{
			Text: "◀️", CallbackData: "previous",
		})
	}
	sliderRow = append(sliderRow, models.InlineKeyboardButton{
		Text: fmt.Sprintf("%d/%d", currentIdx+1, total), CallbackData: "noop",
	})
	if currentIdx < total-1 {
		sliderRow = append(sliderRow, models.InlineKeyboardButton{
			Text: "▶️", CallbackData: "next",
		})
	}
	var controlRow []models.InlineKeyboardButton
	controlRow = append(controlRow, models.InlineKeyboardButton{
		Text: "📁 Скачать файлы", CallbackData: "files",
	})
	switch action {
	case OrderSliderClose:
		controlRow = append(controlRow, models.InlineKeyboardButton{
			Text: "📩 Закрыть", CallbackData: "close",
		})
	case OrderSliderRestore:
		controlRow = append(controlRow, models.InlineKeyboardButton{
			Text: "🔄 Восстановить", CallbackData: "restore",
		})
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, sliderRow, controlRow)
	return keyboard
}

func OrderSliderSelectorKbd(total, currentIdx int) *models.InlineKeyboardMarkup {
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{},
	}
	var sliderRow []models.InlineKeyboardButton
	if currentIdx > 0 {
		sliderRow = append(sliderRow, models.InlineKeyboardButton{
			Text: "◀️", CallbackData: "previous",
		})
	}
	sliderRow = append(sliderRow, models.InlineKeyboardButton{
		Text: fmt.Sprintf("%d/%d", currentIdx+1, total), CallbackData: "noop",
	})
	if currentIdx < total-1 {
		sliderRow = append(sliderRow, models.InlineKeyboardButton{
			Text: "▶️", CallbackData: "next",
		})
	}
	var controlRow []models.InlineKeyboardButton
	controlRow = append(controlRow, models.InlineKeyboardButton{
		Text: "Выбрать", CallbackData: "select",
	})
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, sliderRow, controlRow)
	return keyboard
}
