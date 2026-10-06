package main

import (
	"context"
	"errors"
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

type data struct {
	failS2 bool
}

func main() {
	f := fsm.New(s1, fsm.WithMaxHops[data](64), fsm.WithLogger[data](fsm.NewStdLogger()))

	f.
		On(s1, func(ctx context.Context, data *data) (fsm.State, error) {
			return s2, nil
		}).
		On(s2, func(ctx context.Context, data *data) (fsm.State, error) {
			if data.failS2 {
				return fsm.EmptyState, errors.New("s2 exploded")
			}

			return s3, nil
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

	fmt.Println("Run on happy path")
	_, err := f.Run(context.TODO(), &data{failS2: false})
	if err != nil {
		log.Fatal(err)
	}

	println()

	fmt.Println("Run with fallback")
	_, err = f.Run(context.TODO(), &data{failS2: true})
	if err != nil {
		log.Fatal(err)
	}
}
