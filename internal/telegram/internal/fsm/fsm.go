package fsm

import (
	"sync"
)

type FSM struct {
	states map[int64]State
	mu     *sync.RWMutex
}

type State struct {
	Step ConversationStep
	Data StateData
}

func NewFSM() *FSM {
	return &FSM{
		states: make(map[int64]State),
		mu:     &sync.RWMutex{},
	}
}

func (f *FSM) GetOrCreateState(userID int64) State {
	f.mu.Lock()
	defer f.mu.Unlock()

	state, ok := f.states[userID]
	if !ok {
		state = State{
			Step: StepIdle,
			Data: &IdleData{},
		}
		f.states[userID] = state
	}

	return state
}

func (f *FSM) SetState(userID int64, state State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.states[userID] = state
}

func (f *FSM) SetStep(userID int64, step ConversationStep) {
	state := f.GetOrCreateState(userID)

	state.Step = step
	f.SetState(userID, state)
}

func (f *FSM) UpdateData(userID int64, data StateData) {
	state := f.GetOrCreateState(userID)

	state.Data = data
	f.SetState(userID, state)
}

func (f *FSM) ResetState(userID int64) {
	state := f.GetOrCreateState(userID)

	state.Step = StepIdle
	state.Data = &IdleData{}
	f.SetState(userID, state)
}
