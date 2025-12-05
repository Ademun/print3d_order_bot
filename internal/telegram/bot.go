package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/pkg/config"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/internal/telegram/internal/presentation"

	"github.com/go-telegram/bot"
)

type Bot struct {
	orderService order.Service
	api          *bot.Bot
	router       *fsm.Router
	collector    *media.Collector
}

func NewBot(orderService order.Service, cfg *config.TelegramCfg) (*Bot, *bot.Bot, error) {
	state := fsm.NewFSM()
	router := fsm.NewRouter(state)
	collector := media.NewCollector()
	botOpts := []bot.Option{bot.WithMiddlewares(router.Middleware)}
	b, err := bot.New(cfg.Token, botOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create bot instance: %w", err)
	}

	return &Bot{
		orderService: orderService,
		api:          b,
		router:       router,
		collector:    collector,
	}, b, nil
}

func (b *Bot) Start(ctx context.Context) {
	b.api.RegisterHandler(bot.HandlerTypeMessageText, "orders", bot.MatchTypeCommandStartOnly, b.handleOrderViewCmd)

	b.router.SetAttachmentHandler(b.handleOrderCreation)
	b.router.RegisterHandler(fsm.StepAwaitingOrderType, b.handleOrderType)
	b.router.RegisterHandler(fsm.StepAwaitingClientName, b.handleClientName)
	b.router.RegisterHandler(fsm.StepAwaitingOrderComments, b.handleOrderComments)
	b.router.RegisterHandler(fsm.StepAwaitingNewOrderConfirmation, b.handleNewOrderConfirmation)
	b.router.RegisterHandler(fsm.StepAwaitingOrderSliderAction, b.handleOrderViewAction)

	slog.Info("Started Telegram Bot")
	go b.api.Start(ctx)
}

func (b *Bot) SendMessage(ctx context.Context, params *bot.SendMessageParams) {
	if _, err := b.api.SendMessage(ctx, params); err != nil {
		slog.Error(err.Error())
	}
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
