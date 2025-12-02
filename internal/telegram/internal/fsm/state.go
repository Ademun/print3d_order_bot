package fsm

import "print3d-order-bot/internal/pkg/model"

type ConversationStep int

const (
	StepIdle ConversationStep = iota
	StepAwaitingOrderType
	StepAwaitingClientName
	StepAwaitingOrderComments
	StepAwaitingNewOrderConfirmation
	StepAwaitingOrderID
)

type StateData interface {
	StateData()
}

type IdleData struct{}

func (data *IdleData) StateData() {}

type OrderData struct {
	UserID     int64
	ClientName string
	Comments   *string
	Files      []model.TGOrderFile
	Contacts   []string
	Links      []string
}

func (data *OrderData) StateData() {}

func dataTypeForStep(step ConversationStep) StateData {
	switch step {
	case StepIdle:
		return &IdleData{}
	case StepAwaitingOrderType, StepAwaitingClientName, StepAwaitingOrderComments, StepAwaitingNewOrderConfirmation, StepAwaitingOrderID:
		return &OrderData{}
	}
	return nil
}
