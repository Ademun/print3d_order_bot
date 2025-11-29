package fsm

type ConversationStep int

const (
	StepIdle ConversationStep = iota
)

type StateData interface {
	StateData()
}

type IdleData struct{}

func (data *IdleData) StateData() {}

func dataTypeForStep(step ConversationStep) StateData {
	switch step {
	case StepIdle:
		return &IdleData{}
	}
	return nil
}
