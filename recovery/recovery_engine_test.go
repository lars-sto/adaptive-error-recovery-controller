package recovery

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockStatsSource struct {
	ch <-chan NetworkStats
}

func (m *mockStatsSource) Stats() <-chan NetworkStats { return m.ch }

type countingSink struct {
	calls atomic.Int64
	last  atomic.Value // stores PolicyDecision
}

func (s *countingSink) Publish(d PolicyDecision) {
	s.calls.Add(1)
	s.last.Store(d)
}

type recordingObserver struct {
	calls   atomic.Int64
	changed []bool
	mu      sync.Mutex
}

func (o *recordingObserver) OnSample(_ NetworkStats, _ PolicyDecision, changed bool) {
	o.calls.Add(1)
	o.mu.Lock()
	o.changed = append(o.changed, changed)
	o.mu.Unlock()
}

func (o *recordingObserver) ChangedSeq() []bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]bool, len(o.changed))
	copy(out, o.changed)
	return out
}

type seqController struct {
	mu        sync.Mutex
	decisions []PolicyDecision
	changes   []bool
	i         int
}

func (c *seqController) Decide(_ NetworkStats) (PolicyDecision, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If called more than expected, keep returning last
	if len(c.decisions) == 0 || len(c.changes) == 0 {
		return PolicyDecision{}, false
	}
	if c.i >= len(c.decisions) {
		return c.decisions[len(c.decisions)-1], c.changes[len(c.changes)-1]
	}
	d := c.decisions[c.i]
	ch := c.changes[c.i]
	c.i++
	return d, ch
}

func newTestRecoveryEngine(src StatsSource, sink PolicySink, obs DecisionObserver, ctrl MechanismController) *RecoveryEngine {
	return &RecoveryEngine{
		cfg:         DefaultConfig(),
		src:         src,
		sink:        sink,
		obs:         obs,
		controllers: []MechanismController{ctrl},
	}
}

func TestEngine_PublishOnlyOnChange_ObserverAlwaysCalled(t *testing.T) {
	statsCh := make(chan NetworkStats, 3)
	statsCh <- NetworkStats{LossRate: 0.01, Timestamp: time.Unix(1, 0)}
	statsCh <- NetworkStats{LossRate: 0.02, Timestamp: time.Unix(2, 0)}
	statsCh <- NetworkStats{LossRate: 0.03, Timestamp: time.Unix(3, 0)}
	close(statsCh)

	src := &mockStatsSource{ch: statsCh}
	sink := &countingSink{}
	obs := &recordingObserver{}

	dec1 := NewFlexFECDecision(FlexFECPolicy{
		Enabled:         true,
		Mechanism:       MechanismFlexFEC03,
		NumMediaPackets: 10,
		NumFECPackets:   1,
	})
	dec2 := NewFlexFECDecision(FlexFECPolicy{
		Enabled:         true,
		Mechanism:       MechanismFlexFEC03,
		NumMediaPackets: 10,
		NumFECPackets:   1,
	})
	dec3 := NewFlexFECDecision(FlexFECPolicy{
		Enabled:         true,
		Mechanism:       MechanismFlexFEC03,
		NumMediaPackets: 10,
		NumFECPackets:   2,
	})

	ctrl := &seqController{
		decisions: []PolicyDecision{dec1, dec2, dec3},
		changes:   []bool{true, false, true},
	}

	e := newTestRecoveryEngine(src, sink, obs, ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e.Run(ctx)

	if got := sink.calls.Load(); got != 2 {
		t.Fatalf("sink.Publish calls: got %d, want 2", got)
	}
	if got := obs.calls.Load(); got != 3 {
		t.Fatalf("observer calls: got %d, want 3", got)
	}

	seq := obs.ChangedSeq()
	if len(seq) != 3 || seq[0] != true || seq[1] != false || seq[2] != true {
		t.Fatalf("observer changed sequence: got %#v, want [true false true]", seq)
	}
}

func TestEngine_StopsOnContextDone(t *testing.T) {
	// Channel never closes. engine must stop on ctx cancel
	statsCh := make(chan NetworkStats)
	src := &mockStatsSource{ch: statsCh}

	sink := &countingSink{}
	obs := &recordingObserver{}
	ctrl := &seqController{
		decisions: []PolicyDecision{{
			Mechanism: MechanismNone, Policy: FlexFECPolicy{Enabled: false, Mechanism: MechanismNone}}},
		changes: []bool{false},
	}

	e := newTestRecoveryEngine(src, sink, obs, ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		e.Run(ctx)
	}()

	// Let goroutine block in select
	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// ok
	case <-time.After(250 * time.Millisecond):
		t.Fatal("engine did not stop after context cancellation")
	}
}

func TestEngine_StopsOnClosedStatsChannel(t *testing.T) {
	statsCh := make(chan NetworkStats)
	close(statsCh)

	src := &mockStatsSource{ch: statsCh}
	sink := &countingSink{}
	obs := &recordingObserver{}
	ctrl := &seqController{
		decisions: []PolicyDecision{{Mechanism: MechanismFlexFEC03, Policy: FlexFECPolicy{Enabled: true, Mechanism: MechanismFlexFEC03}}},
		changes:   []bool{true},
	}

	e := newTestRecoveryEngine(src, sink, obs, ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Run(ctx)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(250 * time.Millisecond):
		t.Fatal("engine did not stop after stats channel was closed")
	}

	// No samples were read => no observer calls and no publishes
	if got := sink.calls.Load(); got != 0 {
		t.Fatalf("sink.Publish calls: got %d, want 0", got)
	}
	if got := obs.calls.Load(); got != 0 {
		t.Fatalf("observer calls: got %d, want 0", got)
	}
}
