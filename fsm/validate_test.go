package fsm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/charmingruby/fsm/fsm"
)

type validateData struct{}

func TestFSMValidate(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
		s3 fsm.State = "s3"
	)

	tests := []struct {
		name     string
		initial  fsm.State
		build    func(f *fsm.FSM[validateData])
		opts     []fsm.Option[validateData]
		wantErrs []error
	}{
		{
			name:    "valid machine",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
		},
		{
			name:    "initial terminal without handlers is valid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.Terminal(s1)
			},
		},
		{
			name:     "empty initial is invalid",
			initial:  fsm.EmptyState,
			build:    func(*fsm.FSM[validateData]) {},
			wantErrs: []error{fsm.ErrInvalidInitial},
		},
		{
			name:    "initial without handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s2, func(context.Context, *validateData) (fsm.State, error) {
					return s3, nil
				}).Terminal(s3)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
		{
			name:    "missing terminal is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).On(s2, func(context.Context, *validateData) (fsm.State, error) {
					return s1, nil
				})
			},
			wantErrs: []error{fsm.ErrNoTerminal},
		},
		{
			name:    "non-positive hops are invalid",
			initial: s1,
			opts:    []fsm.Option[validateData]{fsm.WithMaxHops[validateData](0)},
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrInvalidMaxHops},
		},
		{
			name:    "nil handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, nil).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrNilHandler},
		},
		{
			name:    "empty state name is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(fsm.EmptyState, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "empty terminal is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2, fsm.EmptyState)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "self fallback is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).OnFail(s1, s1).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrInvalidFallback},
		},
		{
			name:    "empty fallback is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).OnFail(s1, fsm.EmptyState).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "fallback to target without handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).On(s2, func(context.Context, *validateData) (fsm.State, error) {
					return s3, nil
				}).OnFail(s2, s3).Terminal(s3)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
		{
			name:    "fallback from unknown source is invalid",
			initial: s1,
			build: func(f *fsm.FSM[validateData]) {
				f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
					return s2, nil
				}).OnFail(s3, s2).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []fsm.Option[validateData]
			opts = append(opts, tt.opts...)

			f := fsm.New(tt.initial, opts...)
			tt.build(f)

			err := f.Validate()
			if len(tt.wantErrs) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				for _, want := range tt.wantErrs {
					require.ErrorIs(t, err, want)
				}
			}
		})
	}
}

func TestFSMRevalidatesAfterRegistration(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
	)

	f := fsm.New[validateData](s1)
	f.On(s1, func(context.Context, *validateData) (fsm.State, error) {
		return s2, nil
	})

	require.ErrorIs(t, f.Validate(), fsm.ErrNoTerminal)

	trace, err := f.Run(t.Context(), &validateData{})
	require.ErrorIs(t, err, fsm.ErrNoTransition)
	require.Len(t, trace, 1)
	assert.Equal(t, s1, trace[0].From)
	assert.Equal(t, s2, trace[0].To)

	f.Terminal(s2)

	require.NoError(t, f.Validate())

	trace, err = f.Run(t.Context(), &validateData{})
	require.NoError(t, err)
	require.Len(t, trace, 1)
}
