package main

import (
	"context"
	"fmt"
	"log"

	"github.com/charmingruby/fsm/fsm"
)

const (
	s1 fsm.State = "s1"
	s2 fsm.State = "s2"
	s3 fsm.State = "s3"
	s4 fsm.State = "s4"
	s5 fsm.State = "s5"
	s6 fsm.State = "s6"
	s7 fsm.State = "s7"
)

type data struct{}

func main() {
	f := fsm.New(
		s1,
		fsm.WithMaxHops[data](64),
		fsm.WithLogger[data](fsm.NewStdLogger()),
		fsm.WithGlobalHooks(fsm.GlobalHooks[data]{
			OnEnter: func(ctx context.Context, data *data, state fsm.State) {
				fmt.Printf("[GLOBAL ON ENTER HOOK] data: %+v\n", data)
				fmt.Printf("[GLOBAL ON ENTER HOOK] state: %q\n", state)
			},
			OnExit: func(ctx context.Context, data *data, currentState, nextState fsm.State) {
				fmt.Printf("[GLOBAL ON EXIT HOOK] data: %+v\n", data)
				fmt.Printf("[GLOBAL ON EXIT HOOK] current state: %q\n", currentState)
				fmt.Printf("[GLOBAL ON EXIT HOOK] next state: %q\n", nextState)
			},
			OnTransition: func(ctx context.Context, data *data, hop fsm.Transition) {
				fmt.Printf("[GLOBAL TRANSITION HOOK] hop: %+v\n", hop)
				fmt.Printf("[GLOBAL TRANSITION HOOK] data: %+v\n", data)
			},
		}),
	)

	f.
		On(s1, func(ctx context.Context, data *data) (fsm.State, error) {
			return s2, nil
		}).
		On(s2, func(ctx context.Context, data *data) (fsm.State, error) {
			return s3, nil
		}, fsm.StateHooks[data]{
			OnTransition: func(ctx context.Context, data *data, hop fsm.Transition) {
				fmt.Printf("[s2 TRANSITION HOOK] %+v\n", data)
				fmt.Printf("[s2 TRANSITION HOOK] %+v\n", hop)
			},
		}).
		OnFail(s2, s6).
		On(s3, func(ctx context.Context, data *data) (fsm.State, error) {
			return s4, nil
		}).
		On(s4, func(ctx context.Context, data *data) (fsm.State, error) {
			return s5, nil
		}).
		On(s5, func(ctx context.Context, data *data) (fsm.State, error) {
			return s7, nil
		}).
		On(s6, func(ctx context.Context, data *data) (fsm.State, error) {
			return s7, nil
		}).
		Terminal(s7)

	if err := f.Validate(); err != nil {
		log.Fatal(err)
	}

	_, err := f.Run(context.TODO(), &data{})
	if err != nil {
		log.Fatal(err)
	}
}
