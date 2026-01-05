package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"print3d-order-bot/internal/file"
	"print3d-order-bot/internal/mtproto"
	"print3d-order-bot/internal/order"
	"print3d-order-bot/internal/reconciler"
	"print3d-order-bot/internal/telegram/internal/fsm"
	"print3d-order-bot/internal/telegram/internal/media"
	"print3d-order-bot/pkg/config"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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

	SetupOrderCreationFlow(&OrderCreationDeps{
		Router:       b.router,
		Collector:    b.collector,
		OrderService: b.orderService,
		FileService:  b.fileService,
	})

	SetupOrderViewerFlow(&OrderViewerDeps{
		Router:            b.router,
		OrderService:      b.orderService,
		FileService:       b.fileService,
		ReconcilerService: b.reconcilerService,
		MtprotoClient:     b.mtprotoClient,
	})

	SetupOrderEditFlow(&OrderEditFlowDeps{
		Router:       b.router,
		OrderService: b.orderService,
	})

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
	b.router.Transition(ctx, userID, newStep, newData)
}

func (b *Bot) DownloadFile(ctx context.Context, fileID string, dst io.Writer) error {
	fileInfo, err := b.api.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})

	if fileInfo == nil {
		return err
	}

	if fileInfo.FileSize > 20*1024*1024 {
		return fmt.Errorf("file is too large")
	}

	if err != nil {
		return err
	}

	link := b.api.FileDownloadLink(fileInfo)

	resp, err := http.Get(link)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}

func (b *Bot) UploadFile(ctx context.Context, filename string, file io.ReadCloser, userID int64) error {
	_, err := b.api.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: userID,
		Document: &models.InputFileUpload{
			Filename: filename,
			Data:     file,
		},
	})
	return err
}
