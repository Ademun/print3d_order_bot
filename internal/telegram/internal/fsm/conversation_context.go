package fsm

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ConversationContext[T StateData] struct {
	Ctx    context.Context
	Bot    *bot.Bot
	Update *models.Update
	UserID int64
	Data   T
	router *Router
	step   ConversationStep
}

func (c *ConversationContext[T]) SendMessage(text string, markup *models.ReplyMarkup) error {
	params := &bot.SendMessageParams{
		ChatID:      c.UserID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: markup,
	}
	_, err := c.Bot.SendMessage(c.Ctx, params)
	return err
}

func (c *ConversationContext[T]) Transition(nextStep ConversationStep, data StateData) {
	c.router.Transition(c.Ctx, c.UserID, nextStep, data)
}

func (c *ConversationContext[T]) Complete() {
	c.router.Transition(c.Ctx, c.UserID, StepIdle, &IdleData{})
}
