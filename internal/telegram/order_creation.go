package telegram

import (
	"context"
	"print3d-order-bot/internal/pkg/model"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/internal/telegram/internal/presentation"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleOrderCreation(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	b.collector.ProcessMessage(update.Message, func(window *media.Window) {
		data, ok := state.Data.(*fsm.OrderData)
		if ok {
			newData := &fsm.OrderData{
				UserID:     data.UserID,
				ClientName: data.ClientName,
				Comments:   data.Comments,
				Contacts:   append(data.Contacts, window.Contacts...),
				Links:      append(data.Links, window.Links...),
				Files:      append(data.Files, window.Media...),
			}
			b.tryTransition(ctx, userID, state.Step, newData)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      presentation.AddedDataToOrderMsg(),
				ParseMode: models.ParseModeMarkdown,
			})
			return
		}

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

func (b *Bot) handleOrderType(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	userID := update.CallbackQuery.From.ID
	orderType := update.CallbackQuery.Data

	if orderType == "new_order" {
		b.tryTransition(ctx, userID, fsm.StepAwaitingClientName, state.Data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			Text:      presentation.AskClientNameMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderID, state.Data)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
		Text:      presentation.AskOrderSelectionMsg(),
		ParseMode: models.ParseModeMarkdown,
	})
}

func (b *Bot) handleClientName(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID
	clientName := strings.TrimSpace(update.Message.Text)

	newData, ok := state.Data.(*fsm.OrderData)
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

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderCost, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      presentation.AskOrderCostMsg(),
		ParseMode: models.ParseModeMarkdown,
	})
}

func (b *Bot) handleOrderCost(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	costStr := update.Message.Text
	cost, err := strconv.ParseFloat(costStr, 32)
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepAwaitingOrderCost, state.Data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      presentation.CostValidationErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
	}

	newData, ok := state.Data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
	}
	newData.Cost = float32(cost)

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderComments, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        presentation.AskOrderCommentsMsg(),
		ReplyMarkup: presentation.SkipKbd(),
		ParseMode:   models.ParseModeMarkdown,
	})
}

func (b *Bot) handleOrderComments(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
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
		b.tryTransition(ctx, userID, fsm.StepAwaitingNewOrderConfirmation, state.Data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		})
		return
	}

	comments := strings.TrimSpace(update.Message.Text)

	newData, ok := state.Data.(*fsm.OrderData)
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

func (b *Bot) handleNewOrderConfirmation(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
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

	newData, ok := state.Data.(*fsm.OrderData)
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

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		Text:   presentation.NewOrderCreatedMsg(),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
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
