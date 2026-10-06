package fsm_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/charmingruby/fsm/fsm"
)

type testData struct {
	happyPath bool
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return s3, nil
					}).
					On(s3, func(context.Context, *testData) (fsm.State, error) {
						return s4, nil
					}).
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(_ context.Context, e *testData) (fsm.State, error) {
						if !e.happyPath {
							return fsm.EmptyState, errBoom
						}

						return s3, nil
					}).
					On(s3, func(context.Context, *testData) (fsm.State, error) {
						return s7, nil
					}).
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					OnFail(s2, s6).
					On(s6, func(context.Context, *testData) (fsm.State, error) {
						return s7, nil
					}).
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
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
				f.On(s2, func(context.Context, *testData) (fsm.State, error) {
					return s3, nil
				}).Terminal(s3)
			},
			data:    &testData{},
			ctx:     context.Background,
			wantErr: fsm.ErrNoTransition,
		},
		{
			name:    "no transition registered mid chain",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s3)
			},
			data:     &testData{},
			ctx:      context.Background,
			wantErr:  fsm.ErrNoTransition,
			wantFrom: []fsm.State{s1},
			wantTo:   []fsm.State{s2},
		},
		{
			name:    "fallback to state without handler fails at runtime",
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
			wantFallback: []fsm.State{
				s6,
			},
		},
		{
			name:         "max hops exceeded on self loop",
			initial:      s1,
			maxHops:      3,
			applyMaxHops: true,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s1, nil
				}).Terminal(s2)
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return s3, nil
					}).
					On(s3, func(context.Context, *testData) (fsm.State, error) {
						return s4, nil
					}).
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
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
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
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

			var opts []fsm.Option[testData]
			if tt.applyMaxHops {
				opts = append(opts, fsm.WithMaxHops[testData](tt.maxHops))
			}

			f := fsm.New(tt.initial, opts...)
			tt.build(f)

			// Model the premise: always Validate before Run. Assertions target Run.
			_ = f.Validate()

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

func TestFSMConcurrentRun(t *testing.T) {
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
		// initial is the starting state.
		initial fsm.State
		// wantErr is the expected Run error, if any.
		wantErr error
		// wantErrs are additional expected error matches.
		wantErrs []error
		// wantFrom, wantTo and wantFallback describe the expected trace.
		wantFrom     []fsm.State
		wantTo       []fsm.State
		wantFallback []fsm.State
		// runs is the number of concurrent goroutines.
		runs int
		// mixValidate makes odd goroutines call Validate instead of Run.
		mixValidate bool
	}{
		{
			name:    "concurrent runs happy path",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return s3, nil
					}).
					Terminal(s3)
			},
			runs:     32,
			wantFrom: []fsm.State{s1, s2},
			wantTo:   []fsm.State{s2, s3},
		},
		{
			name:    "concurrent runs with fallback recovery",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).
					On(s2, func(context.Context, *testData) (fsm.State, error) {
						return fsm.EmptyState, errBoom
					}).
					OnFail(s2, s6).
					On(s6, func(context.Context, *testData) (fsm.State, error) {
						return s7, nil
					}).
					Terminal(s7)
			},
			runs:     32,
			wantFrom: []fsm.State{s1, s2, s6},
			wantTo:   []fsm.State{s2, fsm.EmptyState, s7},
			wantFallback: []fsm.State{
				fsm.EmptyState,
				s6,
				fsm.EmptyState,
			},
		},
		{
			name:    "concurrent runs missing initial handler",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s2, func(context.Context, *testData) (fsm.State, error) {
					return s3, nil
				}).Terminal(s3)
			},
			runs:    32,
			wantErr: fsm.ErrNoTransition,
		},
		{
			name:    "concurrent validate and run",
			initial: s1,
			build: func(f *fsm.FSM[testData]) {
				f.On(s1, func(context.Context, *testData) (fsm.State, error) {
					return s2, nil
				}).Terminal(s2)
			},
			runs:        32,
			mixValidate: true,
			wantFrom:    []fsm.State{s1},
			wantTo:      []fsm.State{s2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := fsm.New[testData](tt.initial)
			tt.build(f)
			_ = f.Validate()

			traces := make([][]fsm.Transition, tt.runs)
			errs := make([]error, tt.runs)

			var wg sync.WaitGroup

			for i := range tt.runs {
				wg.Add(1)

				go func(i int) {
					defer wg.Done()

					if tt.mixValidate && i%2 == 1 {
						errs[i] = f.Validate()

						return
					}

					traces[i], errs[i] = f.Run(context.Background(), &testData{})
				}(i)
			}

			wg.Wait()

			for i := range tt.runs {
				if tt.mixValidate && i%2 == 1 {
					require.NoError(t, errs[i])

					continue
				}

				if tt.wantErr == nil {
					require.NoError(t, errs[i])
				} else {
					require.Error(t, errs[i])
					require.ErrorIs(t, errs[i], tt.wantErr)

					for _, want := range tt.wantErrs {
						require.ErrorIs(t, errs[i], want)
					}
				}

				require.Len(t, traces[i], len(tt.wantTo), "unexpected trace: %+v", traces[i])

				var from, to, fallback []fsm.State

				for _, hop := range traces[i] {
					from = append(from, hop.From)
					to = append(to, hop.To)
					fallback = append(fallback, hop.FallbackUsed)
				}

				assert.Equal(t, tt.wantFrom, from, "unexpected From chain")
				assert.Equal(t, tt.wantTo, to, "unexpected To chain")

				if tt.wantFallback != nil {
					assert.Equal(t, tt.wantFallback, fallback, "unexpected FallbackUsed chain")
				}
			}
		})
	}
}
