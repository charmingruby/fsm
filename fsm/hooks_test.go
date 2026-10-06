package fsm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/charmingruby/fsm/fsm"
)

type hooksData struct{}

func TestStateHooks(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
		s3 fsm.State = "s3"
		s6 fsm.State = "s6"
		s7 fsm.State = "s7"
	)

	errBoom := errors.New("boom")

	tests := []struct {
		name string
		// build registers handlers; record appends "global" or "state sX" per call.
		build func(f *fsm.FSM[hooksData], record func(origin string, hop fsm.Transition))
		// ctx overrides the test context when non-nil.
		ctx func() context.Context
		// initial is the starting state.
		initial fsm.State
		// wantErr is the expected Run error, if any.
		wantErr error
		// wantOrder is the expected global/state call order.
		wantOrder []string
		// wantFrom and wantTo describe the expected trace.
		wantFrom []fsm.State
		wantTo   []fsm.State
	}{
		{
			name:    "initial terminal fires nothing",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}, fsm.StateHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).Terminal(s1)
			},
		},
		{
			name:    "state hook fires on single hop after global",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}, fsm.StateHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).Terminal(s2)
			},
			wantOrder: []string{"global s1->s2", "state s1 s1->s2"},
			wantFrom:  []fsm.State{s1},
			wantTo:    []fsm.State{s2},
		},
		{
			name:    "multiple state hooks all fire in order",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				},
					fsm.StateHooks[hooksData]{
						OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
							record("first", hop)
						},
					},
					fsm.StateHooks[hooksData]{
						OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
							record("second", hop)
						},
					}).Terminal(s2)
			},
			wantOrder: []string{"global s1->s2", "first s1->s2", "second s1->s2"},
			wantFrom:  []fsm.State{s1},
			wantTo:    []fsm.State{s2},
		},
		{
			name:    "only attached state hook fires on multiple hops",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}, fsm.StateHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return s3, nil
					}).
					Terminal(s3)
			},
			wantOrder: []string{"global s1->s2", "state s1 s1->s2", "global s2->s3"},
			wantFrom:  []fsm.State{s1, s2},
			wantTo:    []fsm.State{s2, s3},
		},
		{
			name:    "nil and empty state hooks are safe",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], _ func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return s3, nil
					}, fsm.StateHooks[hooksData]{}).
					Terminal(s3)
			},
			wantOrder: []string{"global s1->s2", "global s2->s3"},
			wantFrom:  []fsm.State{s1, s2},
			wantTo:    []fsm.State{s2, s3},
		},
		{
			name:    "failed state hook does not fire but fallback hook does",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}, fsm.StateHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}, fsm.StateHooks[hooksData]{
						OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
							record("state s2", hop)
						},
					}).
					OnFail(s2, s6).
					On(s6, func(context.Context, *hooksData) (fsm.State, error) {
						return s7, nil
					}, fsm.StateHooks[hooksData]{
						OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
							record("state s6", hop)
						},
					}).
					Terminal(s7)
			},
			wantOrder: []string{"global s1->s2", "state s1 s1->s2", "global s6->s7", "state s6 s6->s7"},
			wantFrom:  []fsm.State{s1, s2, s6},
			wantTo:    []fsm.State{s2, fsm.EmptyState, s7},
		},
		{
			name:    "error without fallback fires nothing for failed state",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}, fsm.StateHooks[hooksData]{
						OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
							record("state s2", hop)
						},
					}).
					Terminal(s3)
			},
			wantErr:   errBoom,
			wantOrder: []string{"global s1->s2"},
			wantFrom:  []fsm.State{s1, s2},
			wantTo:    []fsm.State{s2, fsm.EmptyState},
		},
		{
			name:    "canceled context fires nothing",
			initial: s1,
			build: func(f *fsm.FSM[hooksData], record func(string, fsm.Transition)) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}, fsm.StateHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).Terminal(s2)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			var order []string

			record := func(origin string, hop fsm.Transition) {
				order = append(order, origin+" "+string(hop.From)+"->"+string(hop.To))
			}

			f := fsm.New(tt.initial,
				fsm.WithGlobalHooks[hooksData](fsm.GlobalHooks[hooksData]{
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						record("global", hop)
					},
				}),
			)
			tt.build(f, record)

			_ = f.Validate()

			trace, err := f.Run(ctx, &hooksData{})

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			}

			var from, to []fsm.State
			for _, hop := range trace {
				from = append(from, hop.From)
				to = append(to, hop.To)
			}

			assert.Equal(t, tt.wantFrom, from, "unexpected From chain")
			assert.Equal(t, tt.wantTo, to, "unexpected To chain")
			assert.Equal(t, tt.wantOrder, order, "unexpected hook call order")
		})
	}
}

func TestFSMHooks(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
		s3 fsm.State = "s3"
		s6 fsm.State = "s6"
		s7 fsm.State = "s7"
	)

	errBoom := errors.New("boom")

	tests := []struct {
		name string
		// build registers handlers on the already created FSM.
		build func(f *fsm.FSM[hooksData])
		// ctx overrides the test context when non-nil.
		ctx func() context.Context
		// initial is the starting state.
		initial fsm.State
		// wantErr is the expected Run error, if any.
		wantErr error
		// wantEnters is the expected OnEnter state sequence.
		wantEnters []fsm.State
		// wantExits is the expected OnExit from/to pairs.
		wantExits [][2]fsm.State
		// wantHops is the expected OnTransition from/to pairs.
		wantHops [][2]fsm.State
		// wantFrom and wantTo describe the expected trace.
		wantFrom []fsm.State
		wantTo   []fsm.State
	}{
		{
			name:    "initial terminal observes nothing",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s1)
			},
		},
		{
			name:    "happy path observes single hop",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
			wantEnters: []fsm.State{s1},
			wantExits:  [][2]fsm.State{{s1, s2}},
			wantHops:   [][2]fsm.State{{s1, s2}},
			wantFrom:   []fsm.State{s1},
			wantTo:     []fsm.State{s2},
		},
		{
			name:    "happy path observes multiple hops",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return s3, nil
					}).
					Terminal(s3)
			},
			wantEnters: []fsm.State{s1, s2},
			wantExits:  [][2]fsm.State{{s1, s2}, {s2, s3}},
			wantHops:   [][2]fsm.State{{s1, s2}, {s2, s3}},
			wantFrom:   []fsm.State{s1, s2},
			wantTo:     []fsm.State{s2, s3},
		},
		{
			name:    "fallback observes enters but only successful transitions",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *hooksData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					OnFail(s2, s6).
					On(s6, func(context.Context, *hooksData) (fsm.State, error) {
						return s7, nil
					}).
					Terminal(s7)
			},
			wantEnters: []fsm.State{s1, s2, s6},
			wantExits:  [][2]fsm.State{{s1, s2}, {s6, s7}},
			wantHops:   [][2]fsm.State{{s1, s2}, {s6, s7}},
			wantFrom:   []fsm.State{s1, s2, s6},
			wantTo:     []fsm.State{s2, fsm.EmptyState, s7},
		},
		{
			name:    "no transition fails validation before hooks",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s2, func(context.Context, *hooksData) (fsm.State, error) {
					return s3, nil
				}).Terminal(s3)
			},
			wantErr: fsm.ErrNoTransition,
		},
		{
			name:    "canceled context observes nothing",
			initial: s1,
			build: func(f *fsm.FSM[hooksData]) {
				f.On(s1, func(context.Context, *hooksData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			var enters []fsm.State

			var exits [][2]fsm.State

			var hops [][2]fsm.State

			f := fsm.New(tt.initial,
				fsm.WithGlobalHooks[hooksData](fsm.GlobalHooks[hooksData]{
					OnEnter: func(_ context.Context, _ *hooksData, state fsm.State) {
						enters = append(enters, state)
					},
					OnExit: func(_ context.Context, _ *hooksData, from, to fsm.State) {
						exits = append(exits, [2]fsm.State{from, to})
					},
					OnTransition: func(_ context.Context, _ *hooksData, hop fsm.Transition) {
						hops = append(hops, [2]fsm.State{hop.From, hop.To})
					},
				}),
			)
			tt.build(f)

			_ = f.Validate()

			trace, err := f.Run(ctx, &hooksData{})

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			}

			var from, to []fsm.State
			for _, hop := range trace {
				from = append(from, hop.From)
				to = append(to, hop.To)
			}

			assert.Equal(t, tt.wantFrom, from, "unexpected From chain")
			assert.Equal(t, tt.wantTo, to, "unexpected To chain")
			assert.Equal(t, tt.wantEnters, enters, "unexpected OnEnter calls")
			assert.Equal(t, tt.wantExits, exits, "unexpected OnExit calls")
			assert.Equal(t, tt.wantHops, hops, "unexpected OnTransition calls")
		})
	}
}
