package fsm

import (
	"context"
	"errors"
	"log/slog"
	"print3d-order-bot/internal/telegram/internal/media"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Router struct {
	fsm               *FSM
	handlers          map[ConversationStep][]UniversalHandler[StateData]
	pendingUsers      sync.Map
	attachmentHandler UniversalHandler[StateData]
}

func NewRouter(fsm *FSM) *Router {
	return &Router{
		fsm:          fsm,
		handlers:     make(map[ConversationStep][]UniversalHandler[StateData]),
		pendingUsers: sync.Map{},
	}
}

func (r *Router) SetAttachmentHandler(handler UniversalHandler[StateData]) {
	r.attachmentHandler = handler
}

func (r *Router) RegisterHandler(step ConversationStep, handler UniversalHandler[StateData]) {
	r.handlers[step] = append(r.handlers[step], handler)
}

func (r *Router) Middleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		userID := extractUserID(update)
		if userID == 0 {
			return
		}

		if r.tryBlock(userID, ctx, b, update) {
			return
		}

		if isCommand(update) {
			r.fsm.ResetState(userID)
			next(ctx, b, update)
			return
		}

		state := r.fsm.GetOrCreateState(userID)

		var handlers []UniversalHandler[StateData]
		if hasMedia(update) {
			handlers = []UniversalHandler[StateData]{r.attachmentHandler}
		} else {
			hndlrs, exist := r.handlers[state.Step]
			if !exist {
				r.fsm.ResetState(userID)
				next(ctx, b, update)
				return
			}
			handlers = hndlrs
		}

		convCtx := &ConversationContext[StateData]{
			Ctx:    ctx,
			Bot:    b,
			Update: update,
			UserID: userID,
			Data:   state.Data,
			router: r,
			Step:   state.Step,
		}

		for _, handler := range handlers {
			if err := handler(convCtx); err != nil {
				if errors.Is(err, IncompatibleHandler) {
					continue
				}
				slog.Error("Handler error", "error", err, "step", state.Step)
				convCtx.SendMessage("<b>❌ Произошла неизвестная ошибка, попробуйте позже</b>", nil)
				r.fsm.ResetState(userID)
			}
		}
	}
}

func (r *Router) Transition(userID int64, nextStep ConversationStep, data StateData) {
	r.fsm.SetStep(userID, nextStep)
	if data == nil {
		return
	}
	r.fsm.UpdateData(userID, data)
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

func extractUserID(update *models.Update) int64 {
	var userID int64
	if update.Message != nil {
		userID = update.Message.From.ID
	} else if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
	}
	return userID
}

func isCommand(update *models.Update) bool {
	if update.Message == nil {
		return false
	}
	if strings.HasPrefix(update.Message.Text, "/") {
		return true
	}
	return false
}

func hasMedia(update *models.Update) bool {
	if update.Message == nil {
		return false
	}
	return media.HasMedia(update.Message)
}
