package fsm

import (
	"context"
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

func (f *FSM) GetOrCreateState(userID int64) (*State, error) {
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

	return &state, nil
}

func (f *FSM) SetState(userID int64, state State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.states[userID] = state
}

func (f *FSM) SetStep(userID int64, step ConversationStep) error {
	state, err := f.GetOrCreateState(userID)
	if err != nil {
		return err
	}

	state.Step = step
	f.SetState(userID, *state)

	return nil
}

func (f *FSM) UpdateData(ctx context.Context, userID int64, data StateData) error {
	state, err := f.GetOrCreateState(userID)
	if err != nil {
		return err
	}

	state.Data = data
	f.SetState(userID, *state)

	return nil
}

func (f *FSM) ResetState(ctx context.Context, userID int64) error {
	state, err := f.GetOrCreateState(userID)
	if err != nil {
		return err
	}

	state.Step = StepIdle
	state.Data = &IdleData{}
	f.SetState(userID, *state)

	return nil
}
