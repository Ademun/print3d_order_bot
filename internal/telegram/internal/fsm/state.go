package fsm

import "print3d-order-bot/internal/telegram/internal/model"

type ConversationStep int

const (
	StepIdle ConversationStep = iota
	StepAwaitingOrderType
	StepAwaitingClientName
	StepAwaitingOrderCost
	StepAwaitingOrderComments
	StepAwaitingNewOrderConfirmation
	StepAwaitingOrderSelectSliderAction
	StepAwaitingOrderViewSliderAction
	StepAwaitingEditName
	StepAwaitingEditCost
	StepAwaitingEditComments
	StepAwaitingEditOverrideComments
)

type StateData interface {
	StateData()
}

type IdleData struct{}

func (data *IdleData) StateData() {}

type OrderData struct {
	UserID     int64
	ClientName string
	Cost       float32
	Comments   []string
	Contacts   []string
	Links      []string
	Files      []model.File
	OrdersIDs  []int
	CurrentIdx int
}

func (data *OrderData) StateData() {}

type OrderSliderData struct {
	OrdersIDs  []int
	CurrentIdx int
}

func (data *OrderSliderData) StateData() {}

type OrderEditData struct {
	ClientName       *string
	Cost             *float32
	Comments         []string
	OverrideComments *bool
}
