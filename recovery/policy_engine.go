package recovery

import (
	"context"
	"math"
	"time"
)

// Engine calculates FEC policies based on network stats
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

func (e *Engine) Evaluate(s NetworkStats) (PolicyDecision, bool) {
	changed := false
	reasonF := ""

	// 1. Basic protection factor inspired by libwebrtc tables
	targetOverhead := GetLossProtFactor(s.RTTMs, s.LossRate)

	// 2. BWE-Veto (Congestion Control Awareness)
	if s.TargetBitrate > 0 && s.CurrentBitrate > 0 {
		projectedTotal := s.CurrentBitrate * (1.0 + targetOverhead)
		if projectedTotal > s.TargetBitrate {
			// Reduce overhead to prevent congestion
			maxAllowedOverhead := (s.TargetBitrate / s.CurrentBitrate) - 1.0
			if maxAllowedOverhead < 0 {
				maxAllowedOverhead = 0
			}

			if targetOverhead > maxAllowedOverhead {
				targetOverhead = maxAllowedOverhead
				reasonF = "BWE cap: reduced FEC to fit bandwidth"
			}
		}
	}

	// 3. State-Update & Hysteresis
	newEnabledState := targetOverhead > 0.01

	if newEnabledState != e.fecEnabled {
		e.fecEnabled = newEnabledState
		changed = true
		if e.fecEnabled {
			reasonF = "FEC enabled: network protection required"
		} else {
			reasonF = "FEC disabled: stable network"
		}
	}

	// 4. Scheme & Overhead Update
	if e.fecEnabled {
		if e.fecScheme != FECSchemeFlexFEC03 {
			e.fecScheme = FECSchemeFlexFEC03
			changed = true
		}

		// Small threshold so we don't send new policy on every small change
		if math.Abs(e.fecOverhead-targetOverhead) > 0.02 {
			e.fecOverhead = targetOverhead
			changed = true
			if reasonF == "" {
				reasonF = "adjusted protection factor"
			}
		}
	} else {
		if e.fecOverhead != 0 {
			e.fecOverhead = 0
			changed = true
		}
	}

	return e.buildDecision(reasonF, eventTime(s)), changed
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
