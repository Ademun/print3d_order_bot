package telegram

import (
	"context"
	"print3d-order-bot/internal/pkg/model"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleOrderViewCmd(ctx context.Context, api *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	ids, err := b.orderService.GetActiveOrdersIDs(ctx)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	if len(ids) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.EmptyOrderListMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	order, err := b.orderService.GetOrderByID(ctx, ids[0])
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}
	action := extractOrderAction(order.OrderStatus)

	newData := &fsm.OrderSliderData{
		OrdersIDs:  ids,
		CurrentIdx: 0,
	}

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderSliderAction, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(ids), 0, action),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeMarkdown,
	})
}

func (b *Bot) handleOrderViewAction(ctx context.Context, api *bot.Bot, update *models.Update, data fsm.StateData) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	userID := update.CallbackQuery.From.ID

	sliderAction := update.CallbackQuery.Data

	newData, ok := data.(*fsm.OrderSliderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	switch sliderAction {
	case "previous":
		if newData.CurrentIdx > 0 {
			newData.CurrentIdx--
		}
	case "next":
		if newData.CurrentIdx < len(newData.OrdersIDs)-1 {
			newData.CurrentIdx++
		}
	case "close":
		if err := b.orderService.CloseOrder(ctx, newData.OrdersIDs[newData.CurrentIdx]); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.GenericErrorMsg(),
				ParseMode: models.ParseModeMarkdown,
			})
			return
		}
	case "restore":
		if err := b.orderService.RestoreOrder(ctx, newData.OrdersIDs[newData.CurrentIdx]); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.GenericErrorMsg(),
				ParseMode: models.ParseModeMarkdown,
			})
			return
		}
	default:
		return
	}

	order, err := b.orderService.GetOrderByID(ctx, newData.OrdersIDs[newData.CurrentIdx])
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}
	action := extractOrderAction(order.OrderStatus)

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderSliderAction, newData)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      userID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(newData.OrdersIDs), newData.CurrentIdx, action),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeMarkdown,
	})
}

func extractOrderAction(status model.OrderStatus) presentation.OrderSliderAction {
	switch status {
	case model.StatusActive:
		return presentation.OrderSliderClose
	case model.StatusClosed:
		return presentation.OrderSliderRestore
	default:
		return presentation.OrderSliderClose
	}
}
