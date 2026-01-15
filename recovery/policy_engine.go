package recovery

import (
	"context"
	"time"
)

type Engine struct {
	cfg  Config
	src  StatsSource
	sink PolicySink

	// internal state (kept minimal for v1)
	fecEnabled  bool
	fecScheme   FECScheme
	fecOverhead float64
}

func NewEngine(cfg Config, src StatsSource, sink PolicySink) *Engine {
	// init defaults
	if cfg.NACKDisableRTTMs == 0 || cfg.NACKEnableRTTMs == 0 {
		cfg = DefaultConfig()
	}
	return &Engine{
		cfg:         cfg,
		src:         src,
		sink:        sink,
		fecEnabled:  false,         // default: off
		fecScheme:   FECSchemeNone, // default
		fecOverhead: 0.0,
	}
}

// Run consumes stats and emits policy decisions.
// v1 behavior: only emits when a decision changes, and uses the stats timestamp.
func (e *Engine) Run(ctx context.Context) {
	statsCh := e.src.Stats()

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-statsCh:
			if !ok {
				return
			}
			decision, changed := e.Evaluate(s)
			if changed {
				e.sink.Publish(decision)
			}
		}
	}
}

// Evaluate updates internal state based on the provided stats sample,
// and returns the resulting policy decision plus whether anything changed.
func (e *Engine) Evaluate(s NetworkStats) (PolicyDecision, bool) {
	changed := false
	reasonN := ""
	reasonF := ""

	// FEC policy (loss hysteresis + simple overhead mapping)
	if !e.fecEnabled && s.LossRate >= e.cfg.FECEnableLossRate {
		e.fecEnabled = true
		changed = true
		reasonF = "enable FEC: loss high"
	} else if e.fecEnabled && s.LossRate <= e.cfg.FECDisableLossRate {
		e.fecEnabled = false
		if e.fecScheme != FECSchemeNone {
			e.fecScheme = FECSchemeNone
		}
		if e.fecOverhead != 0.0 {
			e.fecOverhead = 0.0
		}
		changed = true
		reasonF = "disable FEC: loss low"
	}

	// If enabled, pick scheme + overhead (placeholder policy).
	if e.fecEnabled {
		// v1: default to FlexFEC03; only mark changed if it differs.
		if e.fecScheme != FECSchemeFlexFEC03 {
			e.fecScheme = FECSchemeFlexFEC03
			changed = true
			if reasonF == "" {
				reasonF = "select FEC scheme"
			}
		}

		// Map loss to overhead in [MinOverhead, MaxOverhead]
		target := clamp(e.cfg.MinOverhead+(s.LossRate*2.0), e.cfg.MinOverhead, e.cfg.MaxOverhead)
		if target != e.fecOverhead {
			e.fecOverhead = target
			changed = true
			if reasonF == "" {
				reasonF = "adjust FEC overhead"
			}
		}
	}

	// If nothing changed, we can still return the current decision (useful in tests),
	// but Run() will only publish on changes.
	return e.buildDecision(joinReasons(reasonN, reasonF), eventTime(s)), changed
}

func (e *Engine) buildDecision(reason string, at time.Time) PolicyDecision {
	return PolicyDecision{
		FEC: FECPolicy{
			Enabled:        e.fecEnabled,
			Scheme:         e.fecScheme,
			TargetOverhead: e.fecOverhead,
			Reason:         reason,
			At:             at,
		},
	}
}

// eventTime uses the stats timestamp if available; otherwise falls back to wall clock.
func eventTime(s NetworkStats) time.Time {
	if !s.Timestamp.IsZero() {
		return s.Timestamp
	}
	return time.Now()
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func joinReasons(a, b string) string {
	switch {
	case a == "" && b == "":
		return "update"
	case a != "" && b == "":
		return a
	case a == "" && b != "":
		return b
	default:
		return a + " | " + b
	}
}
