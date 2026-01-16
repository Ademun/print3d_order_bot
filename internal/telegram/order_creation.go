package telegram

import (
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

type OrderCreationDeps struct {
	Router       *fsm.Router
	Collector    *media.Collector
	OrderService orderSvc.Service
	FileService  fileSvc.Service
}

func SetupOrderCreationFlow(deps *OrderCreationDeps) {
	deps.Router.SetAttachmentHandler(func(ctx *fsm.ConversationContext[fsm.StateData]) error {
		if ctx.Update.Message == nil {
			return nil
		}

		deps.Collector.ProcessMessage(ctx.Update.Message, func(window *media.Window) {
			data, ok := ctx.Data.(*fsm.OrderData)
			var newData *fsm.OrderData

			if ok {
				data.Contacts = append(data.Contacts, window.Contacts...)
				data.Links = append(data.Links, window.Links...)
				data.Files = append(data.Files, window.Media...)

				ctx.Transition(ctx.Step, data)
				if err := ctx.SendMessage(presentation.AddedDataToOrderMsg(), nil); err != nil {
					return
				}
				return
			}

			newData = &fsm.OrderData{
				UserID:   ctx.UserID,
				Files:    window.Media,
				Contacts: window.Contacts,
				Links:    window.Links,
			}
			ctx.Transition(fsm.StepAwaitingOrderType, newData)
			if err := ctx.SendMessage(
				presentation.AskOrderTypeMsg(),
				presentation.OrderTypeKbd(),
			); err != nil {
				return
			}
		})

		return nil
	})

	fsm.Chain[*fsm.OrderData](deps.Router, "order_creation", fsm.StepAwaitingOrderType).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderData], data string) error {
			if data == "new_order" {
				ctx.Transition(fsm.StepAwaitingClientName, ctx.Data)
				return ctx.SendMessage(presentation.AskClientNameMsg(), nil)
			}

			ids, err := deps.OrderService.GetActiveOrdersIDs(ctx.Ctx)
			if err != nil {
				return ctx.Complete(presentation.OrderIDsLoadErrorMsg())
			}

			if len(ids) == 0 {
				return ctx.Complete(presentation.EmptyOrderListMsg())
			}

			ctx.Data.OrdersIDs = ids
			ctx.Data.CurrentIdx = 0

			order, err := deps.OrderService.GetOrderByID(ctx.Ctx, ids[0])
			if err != nil {
				return ctx.Complete(presentation.OrderLoadErrorMsg())
			}

			ctx.Transition(fsm.StepAwaitingOrderSelectSliderAction, ctx.Data)
			if err := ctx.SendMessage(presentation.AskOrderSelectionMsg(), nil); err != nil {
				return err
			}
			return ctx.SendMessage(
				presentation.OrderViewMsg(order),
				presentation.OrderSliderSelectorKbd(len(ids), 0),
			)
		}).
		Then(fsm.StepAwaitingOrderSelectSliderAction).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderData], data string) error {
			switch data {
			case "previous":
				if ctx.Data.CurrentIdx > 0 {
					ctx.Data.CurrentIdx--
				}
				return updateOrderSelector(ctx, deps)

			case "next":
				if ctx.Data.CurrentIdx < len(ctx.Data.OrdersIDs)-1 {
					ctx.Data.CurrentIdx++
				}
				return updateOrderSelector(ctx, deps)

			case "select":
				return finalizeAddToOrder(ctx, deps)

			default:
				return nil
			}
		}).

		// Client name
		Then(fsm.StepAwaitingClientName).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderData], text string) error {
			ctx.Data.ClientName = strings.TrimSpace(text)
			ctx.Transition(fsm.StepAwaitingOrderCost, ctx.Data)
			return ctx.SendMessage(presentation.AskOrderCostMsg(), nil)
		}).

		// Order cost
		Then(fsm.StepAwaitingOrderCost).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderData], text string) error {
			cost, err := presentation.ParseRUB(text)
			if err != nil {
				return ctx.SendMessage(presentation.CostValidationErrorMsg(), nil)
			}

			ctx.Data.Cost = cost
			ctx.Transition(fsm.StepAwaitingOrderComments, ctx.Data)
			return ctx.SendMessage(
				presentation.AskOrderCommentsMsg(),
				presentation.SkipKbd(),
			)
		}).

		// Order comments
		Then(fsm.StepAwaitingOrderComments).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderData], text string) error {
			comments := strings.TrimSpace(text)
			ctx.Data.Comments = []string{comments}

			ctx.Transition(fsm.StepAwaitingNewOrderConfirmation, ctx.Data)
			return ctx.SendMessage(
				presentation.NewOrderPreviewMsg(ctx.Data),
				presentation.YesNoKbd(),
			)
		}).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderData], data string) error {
			if data == "skip" {
				ctx.Data.Comments = make([]string, 0)
				ctx.Transition(fsm.StepAwaitingNewOrderConfirmation, ctx.Data)
				return ctx.SendMessage(
					presentation.NewOrderPreviewMsg(ctx.Data),
					presentation.YesNoKbd(),
				)
			}
			return nil
		}).

		// New order confirmation
		Then(fsm.StepAwaitingNewOrderConfirmation).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderData], data string) error {
			if data == "no" {
				return ctx.Complete(presentation.NewOrderCancelledMsg())
			}

			return finalizeNewOrder(ctx, deps)
		})
}

func updateOrderSelector(ctx *fsm.ConversationContext[*fsm.OrderData], deps *OrderCreationDeps) error {
	order, err := deps.OrderService.GetOrderByID(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx])
	if err != nil {
		return ctx.Complete(presentation.OrderLoadErrorMsg())
	}

	disablePreview := true
	ctx.Transition(fsm.StepAwaitingOrderSelectSliderAction, ctx.Data)
	_, err = ctx.Bot.EditMessageText(ctx.Ctx, &bot.EditMessageTextParams{
		ChatID:      ctx.UserID,
		MessageID:   ctx.Update.CallbackQuery.Message.Message.ID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderSelectorKbd(len(ctx.Data.OrdersIDs), ctx.Data.CurrentIdx),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
	})
	return err
}

func finalizeAddToOrder(ctx *fsm.ConversationContext[*fsm.OrderData], deps *OrderCreationDeps) error {
	deps.Router.Freeze(ctx.UserID, presentation.PendingDownloadMsg())
	defer deps.Router.Unfreeze(ctx.UserID)

	order, err := deps.OrderService.GetOrderByID(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx])
	if err != nil {
		return ctx.Complete(presentation.OrderLoadErrorMsg())
	}

	filesToDownload := make([]fileSvc.RequestFile, len(ctx.Data.Files))
	for i, f := range ctx.Data.Files {
		filesToDownload[i] = fileSvc.RequestFile{
			Name:     f.Name,
			TGFileID: f.TGFileID,
		}
	}

	msgID, _ := ctx.Bot.SendMessage(ctx.Ctx, &bot.SendMessageParams{
		ChatID:    ctx.UserID,
		Text:      presentation.StartingDownloadMsg(len(filesToDownload)),
		ParseMode: models.ParseModeHTML,
	})

	downloaded := deps.FileService.DownloadAndSave(ctx.Ctx, order.FolderPath, filesToDownload)
	orderFiles := make([]orderSvc.File, 0, len(filesToDownload))
	downloadErrors := make(map[string]string)

	for result := range downloaded {
		if result.Err != nil {
			downloadErrors[result.Result.Name] = formatDownloadError(result.Err)
			continue
		}

		orderFiles = append(orderFiles, orderSvc.File{
			Name:     result.Result.Name,
			Checksum: result.Result.Checksum,
			TgFileID: &result.Result.TGFileID,
		})

		if err := ctx.EditMessageText(msgID.ID, presentation.DownloadProgressMsg(result.Result.Name, result.Index, result.Total)); err != nil {
			return err
		}
	}

	if err := ctx.EditMessageText(msgID.ID, presentation.DownloadResultMsg(downloadErrors)); err != nil {
		return err
	}

	if err := deps.OrderService.AddFilesToOrder(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx], orderFiles); err != nil {
		return ctx.Complete(presentation.AddFilesToOrderWarningMsg())
	}

	return ctx.Complete(presentation.AddedDataToOrderMsg())
}

func finalizeNewOrder(ctx *fsm.ConversationContext[*fsm.OrderData], deps *OrderCreationDeps) error {
	deps.Router.Freeze(ctx.UserID, presentation.PendingDownloadMsg())
	defer deps.Router.Unfreeze(ctx.UserID)

	createdAt := time.Now()
	folderPath := CreateFolderPath(ctx.Data.ClientName, createdAt, int(ctx.UserID))

	filesToDownload := make([]fileSvc.RequestFile, len(ctx.Data.Files))
	for i, f := range ctx.Data.Files {
		filesToDownload[i] = fileSvc.RequestFile{
			Name:     f.Name,
			TGFileID: f.TGFileID,
		}
	}

	msgID, _ := ctx.Bot.SendMessage(ctx.Ctx, &bot.SendMessageParams{
		ChatID:    ctx.UserID,
		Text:      presentation.StartingDownloadMsg(len(filesToDownload)),
		ParseMode: models.ParseModeHTML,
	})

	downloaded := deps.FileService.DownloadAndSave(ctx.Ctx, folderPath, filesToDownload)
	orderFiles := make([]orderSvc.File, 0, len(filesToDownload))
	downloadErrors := make(map[string]string)

	for result := range downloaded {
		if result.Err != nil {
			downloadErrors[result.Result.Name] = formatDownloadError(result.Err)
			continue
		}

		orderFiles = append(orderFiles, orderSvc.File{
			Name:     result.Result.Name,
			Checksum: result.Result.Checksum,
			TgFileID: &result.Result.TGFileID,
		})

		slog.Info("res", result)

		if err := ctx.EditMessageText(msgID.ID, presentation.DownloadProgressMsg(result.Result.Name, result.Index, result.Total)); err != nil {
			return err
		}
	}

	if err := ctx.EditMessageText(msgID.ID, presentation.DownloadResultMsg(downloadErrors)); err != nil {
		return err
	}

	data := orderSvc.RequestNewOrder{
		ClientName: ctx.Data.ClientName,
		Cost:       ctx.Data.Cost,
		Comments:   ctx.Data.Comments,
		Contacts:   ctx.Data.Contacts,
		Links:      ctx.Data.Links,
		CreatedAt:  createdAt,
		FolderPath: folderPath,
	}

	if err := deps.OrderService.NewOrder(ctx.Ctx, data, orderFiles); err != nil {
		_ = deps.FileService.DeleteFolder(folderPath)
		return ctx.Complete(presentation.OrderCreationErrorMsg())
	}

	return ctx.Complete(presentation.NewOrderCreatedMsg())
}

func formatDownloadError(err error) string {
	var pathPrepErr *fileSvc.ErrPrepareFilepath
	var downloadErr *fileSvc.ErrDownloadFailed

	switch {
	case errors.Is(err, fileSvc.ErrFileExists):
		return "Файл уже существует"
	case errors.Is(err, fileSvc.ErrCalculateChecksum):
		return "Не удалось проверить целостность файла"
	case errors.As(err, &pathPrepErr):
		return "Не удалось подготовить путь для загрузки файла"
	case errors.As(err, &downloadErr):
		return "Не удалось загрузить файл. Попробуйте уменьшить его размер"
	default:
		return "Неизвестная ошибка. Свяжитесь с разработчиком для устранения"
	}
}
