package presentation

import "github.com/go-telegram/bot/models"

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
