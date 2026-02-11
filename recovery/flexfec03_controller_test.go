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

	cur := 1000.0

	s1 := mkStats(200, 0.20, cur, 1100.0) // cap=0.10
	dec1, changed1 := c.Decide(s1)
	if !changed1 || !dec1.FEC.Enabled {
		t.Fatalf("precondition: expected enabled + changed on first decision")
	}

	s2 := mkStats(200, 0.20, cur, 1110.0) // cap=0.11
	dec2, changed2 := c.Decide(s2)
	if changed2 {
		t.Fatalf("expected changed=false for small overhead delta within deadband")
	}
	_ = dec2

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

func TestFlexFEC03Controller_DisablesWhenBWECapForcesZeroOverhead(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	_, _ = c.Decide(mkStats(200, 0.20, 1000, 2000))

	dec, changed := c.Decide(mkStats(200, 0.20, 1000, 900))

	if !changed {
		t.Fatalf("expected changed=true when BWE cap forces disable")
	}
	if dec.FEC.Enabled {
		t.Fatalf("expected FEC disabled due to zero overhead budget")
	}
	if dec.FEC.TargetOverhead != 0 {
		t.Fatalf("expected overhead=0 after disable")
	}
}

func TestFlexFEC03Controller_NoChangeForIdenticalStats(t *testing.T) {
	// Idempotence
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03

	c := NewFlexFEC03Controller(cfg)

	s := mkStats(200, 0.20, 1000, 2000)

	_, changed1 := c.Decide(s)
	if !changed1 {
		t.Fatalf("expected first decision to change state")
	}

	_, changed2 := c.Decide(s)
	if changed2 {
		t.Fatalf("expected no change for identical stats input")
	}
}

func TestFlexFEC03Controller_DoesNotEnableIfMaxOverheadIsZero(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheme = FECSchemeFlexFEC03
	cfg.MaxOverhead = 0.0 // FEC effectively disabled via config

	c := NewFlexFEC03Controller(cfg)

	dec, changed := c.Decide(mkStats(200, 0.20, 0, 0)) // high loss

	if changed {
		t.Fatalf("expected no state change when max overhead is zero")
	}
	if dec.FEC.Enabled {
		t.Fatalf("expected FEC to remain disabled when max overhead is zero")
	}
	if dec.FEC.TargetOverhead != 0 {
		t.Fatalf("expected overhead=0")
	}
}
