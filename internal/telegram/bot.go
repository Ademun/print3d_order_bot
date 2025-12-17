package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/mtproto"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/reconciler"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
)

type Bot struct {
	orderService      order.Service
	fileService       file.Service
	reconcilerService reconciler.Service
	api               *bot.Bot
	mtprotoClient     *mtproto.Client
	router            *fsm.Router
	collector         *media.Collector
}

func NewBot(orderService order.Service, fileService file.Service, reconcilerService reconciler.Service, mtprotoClient *mtproto.Client, cfg *config.TelegramCfg) (*Bot, error) {
	state := fsm.NewFSM()
	router := fsm.NewRouter(state)
	collector := media.NewCollector()
	botOpts := []bot.Option{bot.WithMiddlewares(router.Middleware)}
	b, err := bot.New(cfg.Token, botOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot instance: %w", err)
	}

	return &Bot{
		orderService:      orderService,
		fileService:       fileService,
		reconcilerService: reconcilerService,
		api:               b,
		mtprotoClient:     mtprotoClient,
		router:            router,
		collector:         collector,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommandStartOnly, b.handlerHelpCmd)
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "orders", bot.MatchTypeCommandStartOnly, b.handleOrderViewCmd)

	b.router.SetAttachmentHandler(b.handleOrderCreation)
	b.router.RegisterHandler(fsm.StepAwaitingOrderType, b.handleOrderType)
	b.router.RegisterHandler(fsm.StepAwaitingOrderSelectSliderAction, b.handleOrderSelectorAction)
	b.router.RegisterHandler(fsm.StepAwaitingClientName, b.handleClientName)
	b.router.RegisterHandler(fsm.StepAwaitingOrderCost, b.handleOrderCost)
	b.router.RegisterHandler(fsm.StepAwaitingOrderComments, b.handleOrderComments)
	b.router.RegisterHandler(fsm.StepAwaitingNewOrderConfirmation, b.handleNewOrderConfirmation)
	b.router.RegisterHandler(fsm.StepAwaitingOrderViewSliderAction, b.handleOrderViewAction)

	slog.Info("Started Telegram Bot")
	go b.api.Start(ctx)
}

func (b *Bot) SendMessage(ctx context.Context, params *bot.SendMessageParams) int {
	msg, err := b.api.SendMessage(ctx, params)
	if err != nil {
		slog.Error("Error sending message", "error", err, "params", params)
		return 0
	}
	return msg.ID
}

func (b *Bot) EditMessageText(ctx context.Context, params *bot.EditMessageTextParams) {
	if _, err := b.api.EditMessageText(ctx, params); err != nil {
		slog.Error(err.Error())
	}
}

func (b *Bot) AnswerCallbackQuery(ctx context.Context, params *bot.AnswerCallbackQueryParams) {
	if _, err := b.api.AnswerCallbackQuery(ctx, params); err != nil {
		slog.Error(err.Error())
	}
}

func (b *Bot) tryTransition(ctx context.Context, userID int64, newStep fsm.ConversationStep, newData fsm.StateData) {
	if err := b.router.Transition(ctx, userID, newStep, newData); err != nil {
		slog.Error(err.Error())
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   presentation.GenericErrorMsg(),
		})
	}
}
