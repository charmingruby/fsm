package main

import (
	"context"
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
	f := fsm.New(s1, fsm.WithMaxHops[data](64))

	f.
		On(s1, func(ctx context.Context, data *data) (fsm.State, error) {
			return s2, nil
		}, nil).
		On(s2, func(ctx context.Context, data *data) (fsm.State, error) {
			return s3, nil
		}, nil).
		OnFail(s2, s6).
		On(s3, func(ctx context.Context, data *data) (fsm.State, error) {
			return s4, nil
		}, nil).
		On(s4, func(ctx context.Context, data *data) (fsm.State, error) {
			return s5, nil
		}, nil).
		On(s5, func(ctx context.Context, data *data) (fsm.State, error) {
			return s7, nil
		}, nil).
		On(s6, func(ctx context.Context, data *data) (fsm.State, error) {
			return s7, nil
		}, nil).
		Terminal(s7)

	_, err := f.Run(context.TODO(), &data{})
	if err != nil {
		log.Fatal(err)
	}
}
