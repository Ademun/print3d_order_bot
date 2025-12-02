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
	}

	currentOrder, err := b.orderService.GetOrderByID(ctx, ids[0])
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.GenericErrorMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
	}
	var orderAction presentation.OrderSliderAction
	if currentOrder.OrderStatus == model.StatusActive {
		orderAction = presentation.OrderSliderClose
	} else {
		orderAction = presentation.OrderSliderRestore
	}

	if len(ids) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    userID,
			Text:      presentation.EmptyOrderListMsg(),
			ParseMode: models.ParseModeMarkdown,
		})
	}

	newData := &fsm.OrderSliderData{
		OrdersIDs:  ids,
		CurrentIdx: 0,
	}
	b.tryTransition(ctx, userID, fsm.StepAwaitingOrderSliderAction, newData)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        presentation.OrderViewMsg(currentOrder),
		ReplyMarkup: presentation.OrderSliderMgmtKbd(len(ids), 0, orderAction),
		ParseMode:   models.ParseModeMarkdown,
	})
}

func (b *Bot) handleOrderViewAction(ctx context.Context, api *bot.Bot, update *models.Update, data *fsm.StateData) {

}
