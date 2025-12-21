package telegram

import (
	"context"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleEditOrderName(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
	}

	newData, ok := state.Data.(*fsm.OrderEditData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	if update.Message != nil {
		newData.ClientName = &update.Message.Text
	}

	b.tryTransition(ctx, userID, fsm.StepAwaitingEditCost, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.AskOrderCostMsg(),
		ReplyMarkup: presentation.SkipKbd(),
		ParseMode:   models.ParseModeHTML,
	})
}
