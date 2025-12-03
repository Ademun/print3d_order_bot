package telegram

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/pkg/model"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/internal/telegram/internal/presentation"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleOrderCreation(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	slog.Info("received update")
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	b.collector.ProcessMessage(update.Message, func(window *media.Window) {
		newData := &fsm.OrderData{
			UserID:   userID,
			Files:    window.Media,
			Contacts: window.Contacts,
			Links:    window.Links,
		}
		b.tryTransition(ctx, userID, fsm.StepAwaitingOrderType, newData)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        presentation.AskOrderTypeMsg(),
			ReplyMarkup: presentation.OrderTypeKbd(),
			ParseMode:   models.ParseModeMarkdown,
		})
	})
}

func (b *Bot) handleOrderType(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	userID := update.CallbackQuery.From.ID
	orderType := update.CallbackQuery.Data

	if orderType == "new_order" {
		b.tryTransition(ctx, userID, fsm.StepAwaitingClientName, data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      presentation.AskClientNameMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderID, data)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		Text:      presentation.AskOrderSelectionMsg(),
		ParseMode: models.ParseModeMarkdown,
	})
}

func (b *Bot) handleClientName(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID
	clientName := strings.TrimSpace(update.Message.Text)

	newData, ok := data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}
	newData.ClientName = clientName

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderComments, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        presentation.AskOrderCommentsMsg(),
		ReplyMarkup: presentation.SkipKbd(),
		ParseMode:   models.ParseModeMarkdown,
	})
}

func (b *Bot) handleOrderComments(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	}

	if shouldSkip(update) {
		b.tryTransition(ctx, userID, fsm.StepAwaitingNewOrderConfirmation, data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		})
		return
	}

	comments := strings.TrimSpace(update.Message.Text)

	newData, ok := data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}
	newData.Comments = append(newData.Comments, comments)

	b.tryTransition(ctx, userID, fsm.StepAwaitingNewOrderConfirmation, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        presentation.NewOrderPreviewMsg(newData),
		ReplyMarkup: presentation.YesNoKbd(),
		ParseMode:   models.ParseModeMarkdown,
	})
}

func (b *Bot) handleNewOrderConfirmation(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	userID := update.CallbackQuery.From.ID
	action := update.CallbackQuery.Data

	if action == "no" {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      presentation.NewOrderCancelledMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	newData, ok := data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
			Text:   presentation.GenericErrorMsg(),
		})
		return
	}

	tgOrder := model.TGOrder{
		ClientName: newData.ClientName,
		Comments:   newData.Comments,
		Contacts:   newData.Contacts,
		Links:      newData.Links,
	}

	if err := b.orderService.NewOrder(ctx, tgOrder, newData.Files); err != nil {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		Text:      presentation.NewOrderCreatedMsg(),
		ParseMode: models.ParseModeMarkdown,
	})
}

func shouldSkip(update *models.Update) bool {
	if update.CallbackQuery == nil {
		return false
	}
	if update.CallbackQuery.Data == "skip" {
		return true
	}
	return false
}
