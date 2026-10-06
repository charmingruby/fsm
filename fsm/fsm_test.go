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
		wantErrs     []error
		build        func(f *fsm.FSM[testData])
		data         *testData
		ctx          func() context.Context
		name         string
		initial      fsm.State
		wantFrom     []fsm.State
		wantTo       []fsm.State
		wantFallback []fsm.State
		maxHops      int
		applyMaxHops bool
	}{
		{
			name:    "initial state is terminal returns empty trace",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.Terminal(s1)
			},
			data: &testData{},
			ctx:  context.Background,
		},
		{
			name:    "initial terminal skips registered handler",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return fsm.EmptyState, errBoom
				}).Terminal(s1)
			},
			data: &testData{},
			ctx:  context.Background,
		},
		{
			name:    "initial terminal wins over canceled context",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.Terminal(s1)
			},
			data: &testData{},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
		},
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
			wantFallback: []fsm.State{
				fsm.EmptyState,
				s6,
				fsm.EmptyState,
			},
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
			wantErrs: []error{fsm.ErrNoFallback},
			wantFrom: []fsm.State{s1, s2},
			wantTo:   []fsm.State{s2, fsm.EmptyState},
			wantFallback: []fsm.State{
				fsm.EmptyState,
				fsm.EmptyState,
			},
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
			name:    "fallback to state without handler fails validation",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return fsm.EmptyState, errBoom
				}).
					OnFail(s1, s6).
					Terminal(s7)
			},
			data:    &testData{},
			ctx:     context.Background,
			wantErr: fsm.ErrNoTransition,
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

				for _, want := range tt.wantErrs {
					require.ErrorIs(t, err, want)
				}
			}

			require.Len(t, trace, len(tt.wantTo), "unexpected trace: %+v", trace)

			var from, to, fallback []fsm.State

			for _, hop := range trace {
				from = append(from, hop.From)
				to = append(to, hop.To)
				fallback = append(fallback, hop.FallbackUsed)
			}

			assert.Equal(t, tt.wantFrom, from, "unexpected From chain")
			assert.Equal(t, tt.wantTo, to, "unexpected To chain")

			if tt.wantFallback != nil {
				assert.Equal(t, tt.wantFallback, fallback, "unexpected FallbackUsed chain")
			}
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
			name:    "initial terminal observes nothing",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s1)
			},
		},
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
			name:    "no transition fails validation before hooks",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s2, okNext(s3)).Terminal(s3)
			},
			wantErr: fsm.ErrNoTransition,
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
				fsm.WithGlobalHooks[testData](fsm.GlobalHooks[testData]{
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

func TestFSMStateHooks(t *testing.T) {
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
		build func(f *fsm.FSM[testData], record func(origin string, hop fsm.Transition))
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
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2), fsm.StateHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).Terminal(s1)
			},
		},
		{
			name:    "state hook fires on single hop after global",
			initial: s1,
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2), fsm.StateHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
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
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2),
					fsm.StateHooks[testData]{
						OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
							record("first", hop)
						},
					},
					fsm.StateHooks[testData]{
						OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
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
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2), fsm.StateHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).
					On(s2, okNext(s3)).
					Terminal(s3)
			},
			wantOrder: []string{"global s1->s2", "state s1 s1->s2", "global s2->s3"},
			wantFrom:  []fsm.State{s1, s2},
			wantTo:    []fsm.State{s2, s3},
		},
		{
			name:    "nil and empty state hooks are safe",
			initial: s1,
			build: func(f *fsm.FSM[testData], _ func(string, fsm.Transition)) {
				f.On(s1, okNext(s2)).
					On(s2, okNext(s3), fsm.StateHooks[testData]{}).
					Terminal(s3)
			},
			wantOrder: []string{"global s1->s2", "global s2->s3"},
			wantFrom:  []fsm.State{s1, s2},
			wantTo:    []fsm.State{s2, s3},
		},
		{
			name:    "failed state hook does not fire but fallback hook does",
			initial: s1,
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2), fsm.StateHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
						record("state s1", hop)
					},
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}, fsm.StateHooks[testData]{
						OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
							record("state s2", hop)
						},
					}).
					OnFail(s2, s6).
					On(s6, okNext(s7), fsm.StateHooks[testData]{
						OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
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
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2)).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}, fsm.StateHooks[testData]{
						OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
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
			build: func(f *fsm.FSM[testData], record func(string, fsm.Transition)) {
				f.On(s1, okNext(s2), fsm.StateHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
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
				fsm.WithLogger[testData](discardLogger{}),
				fsm.WithGlobalHooks[testData](fsm.GlobalHooks[testData]{
					OnTransition: func(_ context.Context, _ *testData, hop fsm.Transition) {
						record("global", hop)
					},
				}),
			)
			tt.build(f, record)

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
			assert.Equal(t, tt.wantOrder, order, "unexpected hook call order")
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
			name:         "zero hops budget is invalid",
			maxHops:      0,
			applyMaxHops: true,
			wantErr:      fsm.ErrInvalidMaxHops,
			wantTraceLen: 0,
		},
		{
			name:         "negative hops budget is invalid",
			maxHops:      -1,
			applyMaxHops: true,
			wantErr:      fsm.ErrInvalidMaxHops,
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
		build    func(f *fsm.FSM[testData])
		opts     []fsm.Option[testData]
		wantErrs []error
	}{
		{
			name:    "valid machine",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
			},
		},
		{
			name:    "initial terminal without handlers is valid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.Terminal(s1)
			},
		},
		{
			name:     "empty initial is invalid",
			initial:  fsm.EmptyState,
			build:    func(*fsm.FSM[testData]) {},
			wantErrs: []error{fsm.ErrInvalidInitial},
		},
		{
			name:    "initial without handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s2, okNext(s3)).Terminal(s3)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
		{
			name:    "missing terminal is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).On(s2, okNext(s1))
			},
			wantErrs: []error{fsm.ErrNoTerminal},
		},
		{
			name:    "non-positive hops are invalid",
			initial: s1,
			opts:    []fsm.Option[testData]{fsm.WithMaxHops[testData](0)},
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrInvalidMaxHops},
		},
		{
			name:    "nil handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, nil).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrNilHandler},
		},
		{
			name:    "empty state name is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(fsm.EmptyState, okNext(s2)).On(s1, okNext(s2)).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "empty terminal is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).Terminal(s2, fsm.EmptyState)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "self fallback is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).OnFail(s1, s1).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrInvalidFallback},
		},
		{
			name:    "empty fallback is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).OnFail(s1, fsm.EmptyState).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrEmptyState},
		},
		{
			name:    "fallback to target without handler is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).On(s2, okNext(s3)).OnFail(s2, s3).Terminal(s3)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
		{
			name:    "fallback from unknown source is invalid",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, okNext(s2)).OnFail(s3, s2).Terminal(s2)
			},
			wantErrs: []error{fsm.ErrNoTransition},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []fsm.Option[testData]{fsm.WithLogger[testData](discardLogger{})}
			opts = append(opts, tt.opts...)

			f := fsm.New(tt.initial, opts...)
			tt.build(f)

			err := f.Validate()
			if len(tt.wantErrs) == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, err, fsm.ErrInvalidFSM)
				for _, want := range tt.wantErrs {
					require.ErrorIs(t, err, want)
				}
			}

			trace, runErr := f.Run(t.Context(), &testData{})
			if len(tt.wantErrs) == 0 {
				require.NoError(t, runErr)
			} else {
				require.Error(t, runErr)
				require.ErrorIs(t, runErr, fsm.ErrInvalidFSM)
				for _, want := range tt.wantErrs {
					require.ErrorIs(t, runErr, want)
				}
				assert.Empty(t, trace)
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

	f := fsm.New(s1, fsm.WithLogger[testData](discardLogger{}))
	f.On(s1, okNext(s2))

	require.ErrorIs(t, f.Validate(), fsm.ErrNoTerminal)

	trace, err := f.Run(t.Context(), &testData{})
	require.ErrorIs(t, err, fsm.ErrInvalidFSM)
	require.ErrorIs(t, err, fsm.ErrNoTerminal)
	assert.Empty(t, trace)

	f.Terminal(s2)

	require.NoError(t, f.Validate())

	trace, err = f.Run(t.Context(), &testData{})
	require.NoError(t, err)
	require.Len(t, trace, 1)
}
