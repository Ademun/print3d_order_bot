package telegram

import (
	"context"
	orderSvc "print3d-order-bot/internal/order"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleOrderViewCmd(ctx context.Context, api *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID

	ids, err := b.orderService.GetActiveOrdersIDs(ctx)
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.OrderIDsLoadErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	if len(ids) == 0 {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.EmptyOrderListMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	order, err := b.orderService.GetOrderByID(ctx, ids[0])
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.OrderLoadErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	action := extractOrderAction(order.Status)

	newData := &fsm.OrderSliderData{
		OrdersIDs:  ids,
		CurrentIdx: 0,
	}

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderViewSliderAction, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(ids), 0, action),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) handleOrderViewAction(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.CallbackQuery == nil {
		return
	}
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	userID := update.CallbackQuery.From.ID

	sliderAction := update.CallbackQuery.Data

	newData, ok := state.Data.(*fsm.OrderSliderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
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
				Text:      presentation.OrderCloseErrorMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}
	case "restore":
		if err := b.orderService.RestoreOrder(ctx, newData.OrdersIDs[newData.CurrentIdx]); err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.OrderRestoreErrorMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}
	case "files":
		b.router.Freeze(userID, presentation.PendingUploadMsg())
		defer b.router.Unfreeze(userID)

		b.reconcilerService.ReconcileOrder(ctx, newData.OrdersIDs[newData.CurrentIdx])
		order, err := b.orderService.GetOrderByID(ctx, newData.OrdersIDs[newData.CurrentIdx])
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.OrderLoadErrorMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}

		files, err := b.fileService.ReadFiles(order.FolderPath)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.FilesLoadErrorMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}

		for file := range files {
			if file.Err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{})
			}
			if err := b.mtprotoClient.UploadFile(ctx, file.Name, file.Body, userID); err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    userID,
					Text:      presentation.UploadErrorMsg(file.Name),
					ParseMode: models.ParseModeHTML,
				})
			}
		}

		return
	case "edit":
		data := &fsm.OrderEditData{
			OrderID: newData.OrdersIDs[newData.CurrentIdx],
		}
		b.tryTransition(ctx, userID, fsm.StepAwaitingEditName, data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      userID,
			Text:        presentation.AskClientNameMsg(),
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: presentation.SkipKbd(),
		})
		return
	default:
		return
	}

	order, err := b.orderService.GetOrderByID(ctx, newData.OrdersIDs[newData.CurrentIdx])
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.OrderLoadErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	action := extractOrderAction(order.Status)

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderViewSliderAction, newData)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      userID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(newData.OrdersIDs), newData.CurrentIdx, action),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
	})
}

func extractOrderAction(status orderSvc.Status) presentation.OrderSliderAction {
	switch status {
	case orderSvc.StatusActive:
		return presentation.OrderSliderClose
	case orderSvc.StatusClosed:
		return presentation.OrderSliderRestore
	default:
		return presentation.OrderSliderClose
	}
}
