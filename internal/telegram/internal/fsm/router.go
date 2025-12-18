package fsm

import (
	"context"
	"log/slog"
	"print3d-order-bot/internal/telegram/internal/media"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type HandlerFunc func(ctx context.Context, api *bot.Bot, update *models.Update, state State)
type Router struct {
	fsm               *FSM
	handlers          map[ConversationStep]HandlerFunc
	pendingUsers      sync.Map
	attachmentHandler HandlerFunc
}

func NewRouter(fsm *FSM) *Router {
	return &Router{
		fsm:          fsm,
		handlers:     make(map[ConversationStep]HandlerFunc),
		pendingUsers: sync.Map{},
	}
}

func (r *Router) SetAttachmentHandler(handler HandlerFunc) {
	r.attachmentHandler = handler
}

func (r *Router) RegisterHandler(step ConversationStep, handler HandlerFunc) {
	r.handlers[step] = handler
}

func (r *Router) Middleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var userID int64
		if update.Message != nil {
			userID = update.Message.From.ID
			if r.tryBlock(userID, ctx, b, update) {
				return
			}
			if strings.HasPrefix(update.Message.Text, "/") {
				if err := r.fsm.ResetState(ctx, userID); err != nil {
					return
				}
				next(ctx, b, update)
				return
			}
			if media.HasMedia(update.Message) {
				state, err := r.fsm.GetOrCreateState(userID)
				if err != nil {
					return
				}
				r.attachmentHandler(ctx, b, update, state)
				return
			}
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else if update.MessageReaction != nil {
			userID = update.MessageReaction.User.ID
		} else {
			return
		}

		if r.tryBlock(userID, ctx, b, update) {
			return
		}

		state, err := r.fsm.GetOrCreateState(userID)
		if err != nil {
			return
		}

		handler, exists := r.handlers[state.Step]

		if exists {
			handler(ctx, b, update, state)
			return
		}

		if err := r.fsm.ResetState(ctx, userID); err != nil {
			return
		}
		next(ctx, b, update)
	}
}

func (r *Router) Transition(ctx context.Context, userID int64, nextStep ConversationStep, data StateData) error {
	if err := r.fsm.SetStep(userID, nextStep); err != nil {
		slog.Error("Failed to update conversation step", "error", err)
		if err := r.fsm.ResetState(ctx, userID); err != nil {
			slog.Error("Fatal error when clearing conversation state", "error", err)
		}
		return err
	}
	if data == nil {
		return nil
	}
	if err := r.fsm.UpdateData(ctx, userID, data); err != nil {
		slog.Error("Failed to update conversation data", "error", err, "service")
		if err := r.fsm.ResetState(ctx, userID); err != nil {
			slog.Error("Fatal error when clearing conversation state", "error", err, "service")
		}
		return err
	}
	return nil
}

func (r *Router) tryBlock(userID int64, ctx context.Context, b *bot.Bot, update *models.Update) bool {
	value, exists := r.pendingUsers.Load(userID)
	if exists {
		msg := value.(string)
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    userID,
				Text:      msg,
				ParseMode: models.ParseModeHTML,
			}); err != nil {
				slog.Error("Failed to send pending message", "error", err, "userID", userID)
			}
		}(ctx, b, update)
	}
	return exists
}

func (r *Router) Freeze(userID int64, msg string) {
	r.pendingUsers.Store(userID, msg)
}

func (r *Router) Unfreeze(userID int64) {
	r.pendingUsers.Delete(userID)
}
