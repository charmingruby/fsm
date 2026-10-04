package main

import (
	"context"
	"errors"
	"log"
	"time"

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

type event struct {
	HappyPath bool
}

func main() {
	f := fsm.New(
		s1,
		fsm.WithMaxHops[event](64),
	)

	f.
		On(s1, func(ctx context.Context, event *event) (fsm.State, error) {
			return s2, nil
		}).
		On(s2, func(ctx context.Context, event *event) (fsm.State, error) {
			if !event.HappyPath {
				return fsm.EmptyState, errors.New("unable to follow to the happy path")
			}

			time.Sleep(1 * time.Second)

			return s3, nil
		}).
		OnFail(s2, s6).
		On(s3, func(ctx context.Context, event *event) (fsm.State, error) {
			return s4, nil
		}).
		On(s4, func(ctx context.Context, event *event) (fsm.State, error) {
			return s5, nil
		}).
		On(s5, func(ctx context.Context, event *event) (fsm.State, error) {
			return s7, nil
		}).
		On(s6, func(ctx context.Context, event *event) (fsm.State, error) {
			return s7, nil
		}).
		Terminal(s7)

	println("Happy Path Exec")

	_, err := f.Trigger(context.TODO(), &event{
		HappyPath: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	println("Exec with fallback")

	_, err = f.Trigger(context.TODO(), &event{
		HappyPath: false,
	})
	if err != nil {
		log.Fatal(err)
	}
}
