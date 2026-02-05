package recovery

import (
	"math"
	"testing"
	"time"
)

// Tests:
//		Bandwidth awareness
//			-
//		Hysteresis
//			- stays activated on small changes
//			- stays deactivated on small changes
//			- deactivates on small change if no bandwidth is available
//		Deadband
//			- stays activated if difference is smaller then config c.cfg.OverheadDeadband
//			- stays deactivated if difference is smaller then config c.cfg.OverheadDeadband

const eps = 1e-6

func mkStats(rtt int, loss float64, cur, target float64) NetworkStats {
	return NetworkStats{
		RTTMs:          rtt,
		LossRate:       loss,
		CurrentBitrate: cur,
		TargetBitrate:  target,
		Timestamp:      time.Unix(123, 0),
	}
}

// Helper: expected max overhead from BWE cap = (target/current) - 1
func bweCap(cur, target float64) float64 {
	if cur <= 0 || target <= 0 {
		return math.Inf(1)
	}
	v := (target / cur) - 1.0
	if v < 0 {
		return 0
	}
	return v
}

func TestFlexFEC03Controller_EnablesWhenLossAboveThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	dec, changed := c.Decide(mkStats(200, 0.10, 0, 0)) // high loss should enable
	if !changed {
		t.Fatalf("expected changed=true on first enable")
	}
	if !dec.FEC.Enabled {
		t.Fatalf("expected FEC enabled")
	}
	if dec.FEC.Scheme != FECSchemeFlexFEC03 {
		t.Fatalf("expected scheme flexfec03, got %q", dec.FEC.Scheme)
	}
}

func TestFlexFEC03Controller_DisablesWhenLossBelowDisableThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	// Enable first
	_, _ = c.Decide(mkStats(200, 0.10, 0, 0))

	// Then drop loss below disable threshold
	dec, changed := c.Decide(mkStats(200, 0.0, 0, 0))
	if !changed {
		t.Fatalf("expected changed=true on disable")
	}
	if dec.FEC.Enabled {
		t.Fatalf("expected FEC disabled")
	}
	if dec.FEC.TargetOverhead != 0 {
		t.Fatalf("expected overhead 0 when disabled, got %v", dec.FEC.TargetOverhead)
	}
}

func TestFlexFEC03Controller_HysteresisKeepsEnabledInBand(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03
	cfg.FECEnableLossRate = 0.03
	cfg.FECDisableLossRate = 0.01

	c := NewFlexFEC03Controller(cfg)

	// Enable
	dec, _ := c.Decide(mkStats(200, 0.10, 0, 0))
	if !dec.FEC.Enabled {
		t.Fatalf("precondition: expected enabled")
	}

	// Loss in hysteresis band: should stay enabled (even if overhead changes)
	dec, _ = c.Decide(mkStats(200, 0.02, 0, 0))
	if !dec.FEC.Enabled {
		t.Fatalf("expected FEC to remain enabled in hysteresis band")
	}
}

func TestFlexFEC03Controller_ClampToMaxOverhead(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03
	cfg.MaxOverhead = 0.10 // intentionally low

	c := NewFlexFEC03Controller(cfg)

	dec, _ := c.Decide(mkStats(400, 0.50, 0, 0))
	if dec.FEC.TargetOverhead > cfg.MaxOverhead+eps {
		t.Fatalf("expected overhead <= %v, got %v", cfg.MaxOverhead, dec.FEC.TargetOverhead)
	}
}

func TestFlexFEC03Controller_BWECapLimitsOverhead(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	// Choose a tight target bitrate so overhead must be capped by BWE.
	cur := 1000.0
	target := 1050.0 // allows ~5% overhead
	s := mkStats(200, 0.20, cur, target)

	dec, _ := c.Decide(s)

	maxAllowed := bweCap(cur, target)
	if dec.FEC.TargetOverhead > maxAllowed+1e-3 { // small float slack
		t.Fatalf("expected overhead <= BWE cap (%v), got %v", maxAllowed, dec.FEC.TargetOverhead)
	}
}

func TestFlexFEC03Controller_DeadbandAvoidsTinyOverheadUpdates(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	// We force overhead to be governed by BWE cap so the test is independent of table details,
	// assuming the table suggests >= cap in these conditions (high loss/RTT).
	cur := 1000.0

	// Step 1: enable + set overhead ~0.10 via BWE cap
	s1 := mkStats(200, 0.20, cur, 1100.0) // cap=0.10
	dec1, changed1 := c.Decide(s1)
	if !changed1 || !dec1.FEC.Enabled {
		t.Fatalf("precondition: expected enabled + changed on first decision")
	}

	// Step 2: slightly different cap (0.11), delta=0.01 < deadband 0.02 => should NOT change
	s2 := mkStats(200, 0.20, cur, 1110.0) // cap=0.11
	dec2, changed2 := c.Decide(s2)
	if changed2 {
		t.Fatalf("expected changed=false for small overhead delta within deadband")
	}
	_ = dec2 // decision should remain stable; we don't assert exact overhead value

	// Step 3: larger cap jump to 0.25, delta >= 0.14 => should change (overhead update)
	s3 := mkStats(200, 0.20, cur, 1250.0) // cap=0.25
	_, changed3 := c.Decide(s3)
	if !changed3 {
		t.Fatalf("expected changed=true for overhead delta exceeding deadband")
	}
}

func TestFlexFEC03Controller_TimestampPropagates(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	ts := time.Unix(999, 0)
	dec, _ := c.Decide(NetworkStats{
		RTTMs:     200,
		LossRate:  0.10,
		Timestamp: ts,
	})
	if !dec.FEC.At.Equal(ts) {
		t.Fatalf("expected At to equal stats timestamp")
	}
}
