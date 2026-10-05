package fsm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/charmingruby/fsm/fsm"
)

type testData struct {
	happyPath bool
}

type discardLogger struct{}

func (discardLogger) Infof(string, ...any) {}

func (discardLogger) Errorf(string, ...any) {}

func okNext(next fsm.State) fsm.StateFunc[testData] {
	return func(context.Context, *testData) (fsm.State, error) {
		return next, nil
	}
}

func TestFSMRun(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
		s3 fsm.State = "s3"
		s4 fsm.State = "s4"
		s6 fsm.State = "s6"
		s7 fsm.State = "s7"
	)

	errBoom := errors.New("boom")

	tests := []struct {
		wantErr      error
		build        func(f *fsm.FSM[testData])
		data         *testData
		ctx          func() context.Context
		name         string
		initial      fsm.State
		wantFrom     []fsm.State
		wantTo       []fsm.State
		maxHops      int
		applyMaxHops bool
	}{
		{
			name:    "happy path single hop to terminal",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantFrom: []fsm.State{s1},
			wantTo:   []fsm.State{s2},
		},
		{
			name:    "happy path multiple hops to terminal",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, okNext(s3)).
					On(s3, okNext(s4)).
					Terminal(s4)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantFrom: []fsm.State{s1, s2, s3},
			wantTo:   []fsm.State{s2, s3, s4},
		},
		{
			name:    "conditional branch happy path",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, func(_ context.Context, e *testData) (fsm.State, error) {
						if !e.happyPath {
							return fsm.EmptyState, errBoom
						}

						return s3, nil
					}).
					On(s3, okNext(s7)).
					Terminal(s7)
			},
			data:     &testData{happyPath: true},
			ctx:      context.Background,
			wantFrom: []fsm.State{s1, s2, s3},
			wantTo:   []fsm.State{s2, s3, s7},
		},
		{
			name:    "fallback recovers to terminal",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					OnFail(s2, s6).
					On(s6, okNext(s7)).
					Terminal(s7)
			},
			data:     &testData{happyPath: false},
			ctx:      context.Background,
			wantFrom: []fsm.State{s1, s2, s6},
			wantTo:   []fsm.State{s2, fsm.EmptyState, s7},
		},
		{
			name:    "error without fallback is returned",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					Terminal(s3)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  errBoom,
			wantFrom: []fsm.State{s1, s2},
			wantTo:   []fsm.State{s2, fsm.EmptyState},
		},
		{
			name:    "no transition registered for initial state",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s2, okNext(s3)).Terminal(s3)
			},
			data:    &testData{},
			ctx:     context.Background,
			wantErr: fsm.ErrNoTransition,
		},
		{
			name:    "no transition registered mid chain",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s3)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  fsm.ErrNoTransition,
			wantFrom: []fsm.State{s1},
			wantTo:   []fsm.State{s2},
		},
		{
			name:    "fallback to state without handler",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return fsm.EmptyState, errBoom
				}).
					OnFail(s1, s6).
					Terminal(s7)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  fsm.ErrNoTransition,
			wantFrom: []fsm.State{s1},
			wantTo:   []fsm.State{fsm.EmptyState},
		},
		{
			name:         "max hops exceeded on self loop",
			initial:      s1,
			maxHops:      3,
			applyMaxHops: true,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s1)).Terminal(s2)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  fsm.ErrMaxHops,
			wantFrom: []fsm.State{s1, s1, s1},
			wantTo:   []fsm.State{s1, s1, s1},
		},
		{
			name:         "max hops exceeded without terminal",
			initial:      s1,
			maxHops:      2,
			applyMaxHops: true,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, okNext(s3)).
					On(s3, okNext(s4)).
					Terminal(s4)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  fsm.ErrMaxHops,
			wantFrom: []fsm.State{s1, s2},
			wantTo:   []fsm.State{s2, s3},
		},
		{
			name:    "context already canceled",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
			},
			data: &testData{},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			wantErr: context.Canceled,
		},
		{
			name:    "context deadline exceeded",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
			},
			data: &testData{},
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
				cancel()

				return ctx
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if tt.ctx != nil {
				ctx = tt.ctx()
			}

			opts := []fsm.Option[testData]{
				fsm.WithLogger[testData](discardLogger{}),
			}
			if tt.applyMaxHops {
				opts = append(opts, fsm.WithMaxHops[testData](tt.maxHops))
			}

			f := fsm.New(tt.initial, opts...)
			tt.build(f)

			trace, err := f.Run(ctx, tt.data)

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			}

			require.Len(t, trace, len(tt.wantTo), "unexpected trace: %+v", trace)

			var from, to []fsm.State
			for _, hop := range trace {
				from = append(from, hop.From)
				to = append(to, hop.To)
			}

			assert.Equal(t, tt.wantFrom, from, "unexpected From chain")
			assert.Equal(t, tt.wantTo, to, "unexpected To chain")
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
		build func(f *fsm.FSM[testData])
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
			name:    "happy path observes single hop",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
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
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, okNext(s3)).
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
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					OnFail(s2, s6).
					On(s6, okNext(s7)).
					Terminal(s7)
			},
			wantEnters: []fsm.State{s1, s2, s6},
			wantExits:  [][2]fsm.State{{s1, s2}, {s6, s7}},
			wantHops:   [][2]fsm.State{{s1, s2}, {s6, s7}},
			wantFrom:   []fsm.State{s1, s2, s6},
			wantTo:     []fsm.State{s2, fsm.EmptyState, s7},
		},
		{
			name:    "no transition observes enter only",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s2, okNext(s3)).Terminal(s3)
			},
			wantErr:    fsm.ErrNoTransition,
			wantEnters: []fsm.State{s1},
		},
		{
			name:    "canceled context observes nothing",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
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
				fsm.WithLogger[testData](discardLogger{}),
				fsm.WithHooks[testData](fsm.Hooks[testData]{
					OnEnter: func(_ context.Context, _ *testData, state fsm.State) {
						enters = append(enters, state)
					},
					OnExit: func(_ context.Context, _ *testData, from, to fsm.State) {
						exits = append(exits, [2]fsm.State{from, to})
					},
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
						hops = append(hops, [2]fsm.State{hop.From, hop.To})
					},
				}),
			)
			tt.build(f)

			trace, err := f.Run(ctx, &testData{})

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

func TestFSMOptions(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
	)

	tests := []struct {
		wantErr      error
		name         string
		maxHops      int
		wantTraceLen int
		applyMaxHops bool
	}{
		{name: "default hops reaches terminal", wantTraceLen: 1},
		{name: "custom max hops still reaches terminal", maxHops: 10, applyMaxHops: true, wantTraceLen: 1},
		{
			name:         "zero hops budget is exceeded",
			maxHops:      0,
			applyMaxHops: true,
			wantErr:      fsm.ErrMaxHops,
			wantTraceLen: 0,
		},
		{
			name:         "custom store does not break Run",
			maxHops:      10,
			applyMaxHops: true,
			wantTraceLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []fsm.Option[testData]{fsm.WithLogger[testData](discardLogger{})}
			if tt.applyMaxHops {
				opts = append(opts, fsm.WithMaxHops[testData](tt.maxHops))
			}

			f := fsm.New(s1, opts...)
			f.On(s1, okNext(s2)).Terminal(s2)

			trace, err := f.Run(t.Context(), &testData{})

			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
			}

			require.Len(t, trace, tt.wantTraceLen)
		})
	}
}

func TestFSMBuilderChaining(t *testing.T) {
	t.Parallel()

	const (
		s1 fsm.State = "s1"
		s2 fsm.State = "s2"
		s3 fsm.State = "s3"
	)

	f := fsm.New[testData](s1, fsm.WithLogger[testData](discardLogger{}))

	require.Same(t, f, f.On(s1, okNext(s2)))
	require.Same(t, f, f.OnFail(s1, s3))
	require.Same(t, f, f.Terminal(s2, s3))

	f.On(s3, okNext(s2))

	trace, err := f.Run(t.Context(), &testData{})

	require.NoError(t, err)
	require.Len(t, trace, 1)
}
