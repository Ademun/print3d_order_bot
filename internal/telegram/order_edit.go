package telegram

import (
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/presentation"
	"strings"
)

type OrderEditFlowDeps struct {
	Router       *fsm.Router
	OrderService order.Service
}

func SetupOrderEditFlow(deps *OrderEditFlowDeps) {
	fsm.Chain[*fsm.OrderEditData](deps.Router, "order_edit", fsm.StepAwaitingEditName).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderEditData], text string) error {
			ctx.Data.ClientName = &text
			ctx.Transition(fsm.StepAwaitingEditCost, ctx.Data)
			return ctx.SendMessage(
				presentation.AskOrderCostMsg(),
				presentation.SkipKbd(),
			)
		}).
		OnCallback(fsm.HandleCallbackWithMessage[*fsm.OrderEditData](
			"skip",
			fsm.StepAwaitingEditCost,
			presentation.AskOrderCostMsg(),
			presentation.SkipKbd(),
		)).
		Then(fsm.StepAwaitingEditCost).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderEditData], text string) error {
			cost, err := presentation.ParseRUB(text)
			if err != nil {
				return ctx.SendMessage(presentation.CostValidationErrorMsg(), nil)
			}

			ctx.Data.Cost = &cost
			ctx.Transition(fsm.StepAwaitingEditComments, ctx.Data)
			return ctx.SendMessage(
				presentation.AskOrderCommentsMsg(),
				presentation.SkipKbd(),
			)
		}).
		OnCallback(fsm.HandleCallbackWithMessage[*fsm.OrderEditData](
			"skip",
			fsm.StepAwaitingEditComments,
			presentation.AskOrderCommentsMsg(),
			presentation.SkipKbd(),
		)).
		Then(fsm.StepAwaitingEditComments).
		OnText(func(ctx *fsm.ConversationContext[*fsm.OrderEditData], text string) error {
			ctx.Data.Comments = strings.Split(text, ".")
			ctx.Transition(fsm.StepAwaitingEditOverrideComments, ctx.Data)
			return ctx.SendMessage(
				presentation.AskOrderCommentsOverrideMsg(),
				presentation.YesNoKbd(),
			)
		}).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderEditData], data string) error {
			if data == "skip" {
				return finalizeOrderEdit(ctx, deps.OrderService)
			}
			return nil
		}).
		Then(fsm.StepAwaitingEditOverrideComments).
		OnCallback(func(ctx *fsm.ConversationContext[*fsm.OrderEditData], data string) error {
			override := data == "yes"
			ctx.Data.OverrideComments = &override
			return finalizeOrderEdit(ctx, deps.OrderService)
		})
}

func finalizeOrderEdit(ctx *fsm.ConversationContext[*fsm.OrderEditData], orderService order.Service) error {
	edit := order.RequestEditOrder{
		ClientName:       ctx.Data.ClientName,
		Cost:             ctx.Data.Cost,
		Comments:         ctx.Data.Comments,
		OverrideComments: ctx.Data.OverrideComments,
	}

	if err := orderService.EditOrder(ctx.Ctx, ctx.Data.OrderID, edit); err != nil {
		return ctx.Complete(presentation.OrderEditErrorMsg())
	}

	return ctx.Complete(presentation.OrderEditedMsg())
}
