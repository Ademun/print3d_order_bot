package telegram

import (
	"context"
	fileSvc "print3d-order-bot/internal/file"
	"print3d-order-bot/internal/mtproto"
	orderSvc "print3d-order-bot/internal/order"
	"print3d-order-bot/internal/reconciler"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleOrderViewCmd(ctx context.Context, api *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID

	ids, err := b.orderService.GetActiveOrdersIDs(ctx)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.OrderIDsLoadErrorMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	if len(ids) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.EmptyOrderListMsg(),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	order, err := b.orderService.GetOrderByID(ctx, ids[0])
	if err != nil {
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

type OrderViewerDeps struct {
	Router            *fsm.Router
	OrderService      orderSvc.Service
	FileService       fileSvc.Service
	ReconcilerService reconciler.Service
	BotApi            *Bot
	MtprotoClient     *mtproto.Client
}

func SetupOrderViewerFlow(deps *OrderViewerDeps) {
	fsm.Chain[*fsm.OrderSliderData](deps.Router, "order_viewer", fsm.StepAwaitingOrderViewSliderAction).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderSliderData], data string) error {
			switch data {
			case "previous":
				if ctx.Data.CurrentIdx > 0 {
					ctx.Data.CurrentIdx--
				}
				return updateOrderView(ctx, deps.OrderService)

			case "next":
				if ctx.Data.CurrentIdx < len(ctx.Data.OrdersIDs)-1 {
					ctx.Data.CurrentIdx++
				}
				return updateOrderView(ctx, deps.OrderService)

			case "close":
				if err := deps.OrderService.CloseOrder(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx]); err != nil {
					return ctx.SendMessage(presentation.OrderCloseErrorMsg(), nil)
				}
				return updateOrderView(ctx, deps.OrderService)

			case "restore":
				if err := deps.OrderService.RestoreOrder(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx]); err != nil {
					return ctx.SendMessage(presentation.OrderRestoreErrorMsg(), nil)
				}
				return updateOrderView(ctx, deps.OrderService)

			case "files":
				return handleOrderFiles(ctx, deps)

			case "edit":
				editData := &fsm.OrderEditData{
					OrderID: ctx.Data.OrdersIDs[ctx.Data.CurrentIdx],
				}
				ctx.Transition(fsm.StepAwaitingEditName, editData)
				return ctx.SendMessage(
					presentation.AskClientNameMsg(),
					presentation.SkipKbd(),
				)

			default:
				return nil
			}
		})
}

func updateOrderView(ctx *fsm.ConversationContext[*fsm.OrderSliderData], orderService orderSvc.Service) error {
	order, err := orderService.GetOrderByID(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx])
	if err != nil {
		return ctx.Complete(presentation.OrderLoadErrorMsg())
	}

	action := extractOrderAction(order.Status)
	disablePreview := true

	ctx.Transition(fsm.StepAwaitingOrderViewSliderAction, ctx.Data)
	_, err = ctx.Bot.EditMessageText(ctx.Ctx, &bot.EditMessageTextParams{
		ChatID:      ctx.UserID,
		MessageID:   ctx.Update.CallbackQuery.Message.Message.ID,
		Text:        presentation.OrderViewMsg(order),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(ctx.Data.OrdersIDs), ctx.Data.CurrentIdx, action),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &disablePreview,
		},
		ParseMode: models.ParseModeHTML,
	})
	return err
}

func handleOrderFiles(ctx *fsm.ConversationContext[*fsm.OrderSliderData], deps *OrderViewerDeps) error {
	deps.Router.Freeze(ctx.UserID, presentation.PendingUploadMsg())
	defer deps.Router.Unfreeze(ctx.UserID)

	deps.ReconcilerService.ReconcileOrder(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx])

	order, err := deps.OrderService.GetOrderByID(ctx.Ctx, ctx.Data.OrdersIDs[ctx.Data.CurrentIdx])
	if err != nil {
		return ctx.SendMessage(presentation.OrderLoadErrorMsg(), nil)
	}

	files, err := deps.FileService.ReadFiles(order.FolderPath)
	if err != nil {
		return ctx.SendMessage(presentation.FilesLoadErrorMsg(), nil)
	}

	for file := range files {
		if file.Err != nil {
			if err := ctx.SendMessage(presentation.FilesLoadErrorMsg(), nil); err != nil {
				return err
			}
			continue
		}

		var uploadErr error
		if file.Size <= 49*1024*1024 {
			uploadErr = deps.BotApi.UploadFile(ctx.Ctx, file.Name, file.Body, ctx.UserID)
		} else {
			uploadErr = deps.MtprotoClient.UploadFile(ctx.Ctx, file.Name, file.Body, ctx.UserID)
		}

		if uploadErr != nil {
			if err := ctx.SendMessage(presentation.UploadErrorMsg(file.Name), nil); err != nil {
				return err
			}
		}
	}

	return nil
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
