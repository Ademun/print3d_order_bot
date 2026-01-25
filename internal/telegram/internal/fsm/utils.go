package fsm

import "github.com/go-telegram/bot/models"

func HandleCallbackWithMessage[T StateData](
	callback string,
	nextStep ConversationStep,
	askMessage string,
	markup models.ReplyMarkup,
) func(*ConversationContext[T], string) error {
	return func(ctx *ConversationContext[T], data string) error {
		if err := ctx.AnswerCallbackQuery("", false); err != nil {
			return err
		}
		if data == callback {
			ctx.Transition(nextStep, ctx.Data)
			return ctx.SendMessage(askMessage, markup)
		}
		return nil
	}
}
