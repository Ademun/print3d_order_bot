package fsm

import "print3d-order-bot/internal/telegram/internal/model"

type ConversationStep int

const (
	StepIdle ConversationStep = iota
	StepAwaitingOrderType
	StepAwaitingPrintType
	StepAwaitingClientName
	StepAwaitingOrderCost
	StepAwaitingOrderComments
	StepAwaitingNewOrderConfirmation
	StepAwaitingOrderSelectSliderAction
	StepAwaitingOrderViewSliderAction
	StepAwaitingEditPrintType
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
	PrintType  string
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
	OrderID          int
	PrintType        *string
	ClientName       *string
	Cost             *float32
	Comments         []string
	OverrideComments *bool
}

func (data *OrderEditData) StateData() {}
