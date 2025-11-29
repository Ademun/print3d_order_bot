package fsm

import (
	"context"
	"fmt"
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

func (f *FSM) GetState(userID int64) (*State, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	state, ok := f.states[userID]
	if !ok {
		return nil, fmt.Errorf("user %d not found", userID)
	}

	return &state, nil
}

func (f *FSM) SetState(userID int64, state State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.states[userID] = state
}

func (f *FSM) SetStep(userID int64, step ConversationStep) error {
	state, err := f.GetState(userID)
	if err != nil {
		return err
	}

	state.Step = step
	f.SetState(userID, *state)

	return nil
}

func (f *FSM) UpdateData(ctx context.Context, userID int64, data StateData) error {
	state, err := f.GetState(userID)
	if err != nil {
		return err
	}

	state.Data = data
	f.SetState(userID, *state)

	return nil
}

func (f *FSM) ResetState(ctx context.Context, userID int64) error {
	state, err := f.GetState(userID)
	if err != nil {
		return err
	}

	state.Step = StepIdle
	state.Data = &IdleData{}
	f.SetState(userID, *state)

	return nil
}
