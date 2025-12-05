package fsm

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type HandlerFunc func(ctx context.Context, api *bot.Bot, update *models.Update, state StateData)
type Router struct {
	fsm      *FSM
	handlers map[ConversationStep]HandlerFunc
	mu       *sync.RWMutex
}

func NewRouter(fsm *FSM) *Router {
	return &Router{
		fsm:      fsm,
		handlers: make(map[ConversationStep]HandlerFunc),
		mu:       &sync.RWMutex{},
	}
}

func (r *Router) RegisterHandler(step ConversationStep, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[step] = handler
}

func (r *Router) Middleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var userID int64
		if update.Message != nil {
			userID = update.Message.From.ID
			if strings.HasPrefix(update.Message.Text, "/") {
				if err := r.fsm.ResetState(ctx, userID); err != nil {
					return
				}
				next(ctx, b, update)
				return
			}
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
		} else if update.MessageReaction != nil {
			userID = update.MessageReaction.User.ID
		} else {
			return
		}

		state, err := r.fsm.GetOrCreateState(userID)
		if err != nil {
			return
		}

		r.mu.RLock()
		handler, exists := r.handlers[state.Step]
		r.mu.RUnlock()

		if exists {
			handler(ctx, b, update, state.Data)
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
			slog.Error("Fatal redis error when clearing conversation state", "error", err)
		}
		return err
	}
	if data == nil {
		return nil
	}
	if err := r.fsm.UpdateData(ctx, userID, data); err != nil {
		slog.Error("Failed to update conversation data", "error", err, "service")
		if err := r.fsm.ResetState(ctx, userID); err != nil {
			slog.Error("Fatal redis error when clearing conversation state", "error", err, "service")
		}
		return err
	}
	return nil
}
