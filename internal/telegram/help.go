package telegram

import (
	"context"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handlerHelpCmd(ctx context.Context, api *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.From.ID,
		Text:      presentation.HelpMsg(),
		ParseMode: models.ParseModeMarkdown,
	})
}
