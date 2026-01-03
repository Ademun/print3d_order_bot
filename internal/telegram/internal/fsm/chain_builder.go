package fsm

import (
	"errors"
	"fmt"
)

var IncompatibleHandler error = errors.New("incompatible handler")

type UniversalHandler[T StateData] func(*ConversationContext[T]) error

type TextHandler[T StateData] func(*ConversationContext[T], string) error

type CallbackHandler[T StateData] func(*ConversationContext[T], string) error

func Chain[T StateData](router *Router, name string, initialStep ConversationStep) *ChainDefinition[T] {
	return &ChainDefinition[T]{
		name:    name,
		router:  router,
		current: initialStep,
	}
}

type ChainDefinition[T StateData] struct {
	name    string
	router  *Router
	current ConversationStep
}

func (c *ChainDefinition[T]) OnText(handler TextHandler[T]) *ChainDefinition[T] {
	wrapped := func(ctx *ConversationContext[StateData]) error {
		typedData, ok := ctx.Data.(T)
		if !ok {
			return fmt.Errorf("type assertion failed")
		}
		typedCtx := &ConversationContext[T]{
			Ctx:    ctx.Ctx,
			Bot:    ctx.Bot,
			Update: ctx.Update,
			UserID: ctx.UserID,
			Data:   typedData,
			router: ctx.router,
			step:   ctx.step,
		}

		if typedCtx.Update.Message == nil {
			return IncompatibleHandler
		}

		return handler(typedCtx, typedCtx.Update.Message.Text)
	}

	c.router.RegisterHandler(c.current, wrapped)
	return c
}

func (c *ChainDefinition[T]) OnCallback(handler CallbackHandler[T]) *ChainDefinition[T] {
	wrapped := func(ctx *ConversationContext[StateData]) error {
		typedData, ok := ctx.Data.(T)
		if !ok {
			return fmt.Errorf("type assertion failed")
		}
		typedCtx := &ConversationContext[T]{
			Ctx:    ctx.Ctx,
			Bot:    ctx.Bot,
			Update: ctx.Update,
			UserID: ctx.UserID,
			Data:   typedData,
			router: ctx.router,
			step:   ctx.step,
		}

		if typedCtx.Update.CallbackQuery == nil {
			return IncompatibleHandler
		}

		return handler(typedCtx, typedCtx.Update.CallbackQuery.Data)
	}

	c.router.RegisterHandler(c.current, wrapped)
	return c
}

func (c *ChainDefinition[T]) Then(nextStep ConversationStep) *ChainDefinition[T] {
	c.current = nextStep
	return c
}
