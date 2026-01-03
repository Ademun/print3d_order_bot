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

func (c *ConversationContext[T]) SendMessage(text string, markup models.ReplyMarkup) error {
	if c.Update.CallbackQuery != nil {
		if err := c.AnswerCallbackQuery("", false); err != nil {
			return err
		}
	}
	params := &bot.SendMessageParams{
		ChatID:      c.UserID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: markup,
	}
	_, err := c.Bot.SendMessage(c.Ctx, params)
	return err
}

func (c *ConversationContext[T]) AnswerCallbackQuery(text string, showAlert bool) error {
	if c.Update.CallbackQuery == nil {
		return nil
	}

	_, err := c.Bot.AnswerCallbackQuery(c.Ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: c.Update.CallbackQuery.ID,
		Text:            text,
		ShowAlert:       showAlert,
	})
	return err
}

func (c *ConversationContext[T]) EditMessageText(messageID int, text string) error {
	if c.Update.CallbackQuery != nil {
		if err := c.AnswerCallbackQuery("", false); err != nil {
			return err
		}
	}

	_, err := c.Bot.EditMessageText(c.Ctx, &bot.EditMessageTextParams{
		ChatID:    c.UserID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	return err
}

func (c *ConversationContext[T]) EditMessageReplyMarkup(messageID int, markup models.ReplyMarkup) error {
	if c.Update.CallbackQuery != nil {
		if err := c.AnswerCallbackQuery("", false); err != nil {
			return err
		}
	}

	_, err := c.Bot.EditMessageReplyMarkup(c.Ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      c.UserID,
		MessageID:   messageID,
		ReplyMarkup: markup,
	})
	return err
}

func (c *ConversationContext[T]) Transition(nextStep ConversationStep, data StateData) {
	c.router.Transition(c.Ctx, c.UserID, nextStep, data)
}

func (c *ConversationContext[T]) Complete(text string) error {
	c.router.Transition(c.Ctx, c.UserID, StepIdle, &IdleData{})
	return c.SendMessage(text, nil)
}
