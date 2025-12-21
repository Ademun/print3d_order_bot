package telegram

import (
	"context"
	"errors"
	"log/slog"
	fileSvc "print3d-order-bot/internal/file"
	orderSvc "print3d-order-bot/internal/order"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/internal/telegram/internal/presentation"
	"strings"
	"time"

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
				ChatID:    userID,
				Text:      presentation.AddedDataToOrderMsg(),
				ParseMode: models.ParseModeHTML,
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
			ChatID:      userID,
			Text:        presentation.AskOrderTypeMsg(),
			ReplyMarkup: presentation.OrderTypeKbd(),
			ParseMode:   models.ParseModeHTML,
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
			ChatID:    userID,
			Text:      presentation.AskClientNameMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	newData, ok := state.Data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

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

	newData.OrdersIDs = ids
	newData.CurrentIdx = 0

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

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderSelectSliderAction, state.Data)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      presentation.AskOrderSelectionMsg(),
		ParseMode: models.ParseModeHTML,
	})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderSelectorKbd(len(ids), 0),
		ParseMode:   models.ParseModeHTML,
	})
}

func (b *Bot) handleOrderSelectorAction(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.CallbackQuery == nil {
		return
	}
	userID := update.CallbackQuery.From.ID

	sliderAction := update.CallbackQuery.Data

	newData, ok := state.Data.(*fsm.OrderData)
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
	case "select":
		b.router.Freeze(userID, presentation.PendingDownloadMsg())
		defer b.router.Unfreeze(userID)
		filesToDownload := make([]fileSvc.RequestFile, len(newData.Files))
		for i, f := range newData.Files {
			filesToDownload[i] = fileSvc.RequestFile{
				Name:     f.Name,
				TGFileID: f.TGFileID,
			}
		}

		msgID := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StartingDownloadMsg(len(filesToDownload)),
			ParseMode: models.ParseModeHTML,
		})

		order, err := b.orderService.GetOrderByID(ctx, msgID)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.OrderLoadErrorMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}

		downloaded := b.fileService.DownloadAndSave(ctx, order.FolderPath, filesToDownload)
		orderFiles := make([]orderSvc.File, 0, len(filesToDownload))
		downloadErrors := make(map[string]string)
		for result := range downloaded {
			if result.Err != nil {
				var err string
				var pathPrepErr *fileSvc.ErrPrepareFilepath
				var downloadErr *fileSvc.ErrDownloadFailed
				if errors.Is(result.Err, fileSvc.ErrFileExists) {
					err = "Файл уже существует"
				} else if errors.Is(result.Err, fileSvc.ErrCalculateChecksum) {
					err = "Не удалось проверить целостность файла"
				} else if errors.As(result.Err, &pathPrepErr) {
					err = "Не удалось подготовить путь для загрузки файла"
				} else if errors.As(result.Err, &downloadErr) {
					err = "Не удалось загрузить файл. Попробуйте уменьшить его размер"
				} else {
					err = "Неизвестная ошибка. Свяжитесь с разработчиком для устранения"
				}
				downloadErrors[result.Result.Name] = err

				continue
			}
			orderFiles = append(orderFiles, orderSvc.File{
				Name:     result.Result.Name,
				Checksum: result.Result.Checksum,
				TgFileID: &result.Result.TGFileID,
			})
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    userID,
				MessageID: msgID,
				Text:      presentation.DownloadProgressMsg(result.Result.Name, result.Index, result.Total),
				ParseMode: models.ParseModeHTML,
			})
		}
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    userID,
			MessageID: msgID,
			Text:      presentation.DownloadResultMsg(downloadErrors),
			ParseMode: models.ParseModeHTML,
		})
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		if err := b.orderService.AddFilesToOrder(ctx, newData.OrdersIDs[newData.CurrentIdx], orderFiles); err != nil {
			b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      presentation.AddFilesToOrderWarningMsg(),
				ParseMode: models.ParseModeHTML,
			})
			return
		}
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.AddedDataToOrderMsg(),
			ParseMode: models.ParseModeHTML,
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

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderSelectSliderAction, newData)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      userID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderSelectorKbd(len(newData.OrdersIDs), newData.CurrentIdx),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
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
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	newData.ClientName = clientName

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderCost, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      presentation.AskOrderCostMsg(),
		ParseMode: models.ParseModeHTML,
	})
}

func (b *Bot) handleOrderCost(ctx context.Context, api *bot.Bot, update *models.Update, state fsm.State) {
	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID

	costStr := update.Message.Text
	cost, err := presentation.ParseRUB(costStr)
	if err != nil {
		b.tryTransition(ctx, userID, fsm.StepAwaitingOrderCost, state.Data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.CostValidationErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	newData, ok := state.Data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	newData.Cost = cost

	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderComments, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.AskOrderCommentsMsg(),
		ReplyMarkup: presentation.SkipKbd(),
		ParseMode:   models.ParseModeHTML,
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
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
	}

	newData, ok := state.Data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	newData.Comments = make([]string, 0)

	if shouldSkip(update) {
		b.tryTransition(ctx, userID, fsm.StepAwaitingNewOrderConfirmation, state.Data)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      userID,
			Text:        presentation.NewOrderPreviewMsg(newData),
			ReplyMarkup: presentation.YesNoKbd(),
			ParseMode:   models.ParseModeHTML,
		})
		return
	}

	comments := strings.TrimSpace(update.Message.Text)
	newData.Comments = append(newData.Comments, comments)

	b.tryTransition(ctx, userID, fsm.StepAwaitingNewOrderConfirmation, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.NewOrderPreviewMsg(newData),
		ReplyMarkup: presentation.YesNoKbd(),
		ParseMode:   models.ParseModeHTML,
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
			ChatID:    userID,
			Text:      presentation.NewOrderCancelledMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}
	b.router.Freeze(userID, presentation.PendingDownloadMsg())
	defer b.router.Unfreeze(userID)

	newData, ok := state.Data.(*fsm.OrderData)
	if !ok {
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.StateConversionErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	createdAt := time.Now()
	folderPath := CreateFolderPath(newData.ClientName, createdAt, update.CallbackQuery.Message.Message.ID)

	filesToDownload := make([]fileSvc.RequestFile, len(newData.Files))
	for i, f := range newData.Files {
		filesToDownload[i] = fileSvc.RequestFile{
			Name:     f.Name,
			TGFileID: f.TGFileID,
		}
	}

	msgID := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    userID,
		Text:      presentation.StartingDownloadMsg(len(filesToDownload)),
		ParseMode: models.ParseModeHTML,
	})

	downloaded := b.fileService.DownloadAndSave(ctx, folderPath, filesToDownload)
	orderFiles := make([]orderSvc.File, 0, len(filesToDownload))
	downloadErrors := make(map[string]string)
	for result := range downloaded {
		if result.Err != nil {
			var err string
			var pathPrepErr *fileSvc.ErrPrepareFilepath
			var downloadErr *fileSvc.ErrDownloadFailed
			if errors.Is(result.Err, fileSvc.ErrFileExists) {
				err = "Файл уже существует"
			} else if errors.Is(result.Err, fileSvc.ErrCalculateChecksum) {
				err = "Не удалось проверить целостность файла"
			} else if errors.As(result.Err, &pathPrepErr) {
				err = "Не удалось подготовить путь для загрузки файла"
			} else if errors.As(result.Err, &downloadErr) {
				err = "Не удалось загрузить файл. Попробуйте уменьшить его размер"
			} else {
				err = "Неизвестная ошибка. Свяжитесь с разработчиком для устранения"
			}
			downloadErrors[result.Result.Name] = err

			continue
		}
		orderFiles = append(orderFiles, orderSvc.File{
			Name:     result.Result.Name,
			Checksum: result.Result.Checksum,
			TgFileID: &result.Result.TGFileID,
		})
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    userID,
			MessageID: msgID,
			Text:      presentation.DownloadProgressMsg(result.Result.Name, result.Index, result.Total),
			ParseMode: models.ParseModeHTML,
		})
	}
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    userID,
		MessageID: msgID,
		Text:      presentation.DownloadResultMsg(downloadErrors),
		ParseMode: models.ParseModeHTML,
	})

	data := orderSvc.RequestNewOrder{
		ClientName: newData.ClientName,
		Cost:       newData.Cost,
		Comments:   newData.Comments,
		Contacts:   newData.Contacts,
		Links:      newData.Links,
		CreatedAt:  createdAt,
		FolderPath: folderPath,
	}

	if err := b.orderService.NewOrder(ctx, data, orderFiles); err != nil {
		if err := b.fileService.DeleteFolder(folderPath); err != nil {
			slog.Error(err.Error())
		}
		b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.OrderCreationErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	disablePreview := true
	b.tryTransition(ctx, userID, fsm.StepIdle, &fsm.IdleData{})
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userID,
		Text:   presentation.NewOrderCreatedMsg(),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
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
