package recovery

import (
	"math"
)

const (
	OverheadDeadband = 0.02
)

// FlexFEC03Controller implements the v1 policy logic for FlexFEC-03.
// It keeps minimal internal state for hysteresis and deadband updates.
type FlexFEC03Controller struct {
	cfg Config

	enabled  bool
	scheme   FECScheme
	overhead float64
}

func NewFlexFEC03Controller(cfg Config) *FlexFEC03Controller {
	return &FlexFEC03Controller{
		cfg:      cfg,
		enabled:  false,
		scheme:   FECSchemeFlexFEC03, // invariant for this controller
		overhead: 0.0,
	}
}

func (c *FlexFEC03Controller) Decide(s NetworkStats) (PolicyDecision, bool) {
	changed := false
	reason := ""

	// 1) Compute baseline protection overhead from RTT + loss (interpolated table)
	targetOverhead := GetLossProtFactor(s.RTTMs, s.LossRate)

	// 2) Clamp to feasible bounds from config
	targetOverhead = clamp(targetOverhead, c.cfg.MinOverhead, c.cfg.MaxOverhead)

	// 3) Bandwidth-awareness (BWE veto):
	// If FEC would exceed the target bitrate, cap overhead to fit the budget.
	if s.TargetBitrate > 0 && s.CurrentBitrate > 0 {
		projectedTotal := s.CurrentBitrate * (1.0 + targetOverhead)
		if projectedTotal > s.TargetBitrate {
			maxAllowedOverhead := (s.TargetBitrate / s.CurrentBitrate) - 1.0
			if maxAllowedOverhead < 0 {
				maxAllowedOverhead = 0
			}
			maxAllowedOverhead = clamp(maxAllowedOverhead, c.cfg.MinOverhead, c.cfg.MaxOverhead)

			if targetOverhead > maxAllowedOverhead {
				targetOverhead = maxAllowedOverhead
				reason = joinReasons(reason, "BWE cap: reduced FEC to fit bandwidth")
			}
		}
	}

	// 4) Enable/disable hysteresis based on loss thresholds.
	newEnabled := c.enabled
	if !c.enabled {
		// lossRate >= c.cfg.FECEnableLossRate && targetOverhead > 0
		if c.shouldEnable(s.LossRate, targetOverhead) {
			newEnabled = true
		}
	} else {
		// lossRate <= c.cfg.FECDisableLossRate || targetOverhead <= 0
		if c.shouldDisable(s.LossRate, targetOverhead) {
			newEnabled = false
		}
	}

	if newEnabled != c.enabled {
		c.enabled = newEnabled
		changed = true
		if c.enabled {
			reason = joinReasons(reason, "FEC enabled: network protection required")
		} else {
			reason = joinReasons(reason, "FEC disabled: stable network")
		}
	}

	// 5) Overhead update with deadband (avoid updates on tiny fluctuations).
	if c.enabled {
		if math.Abs(c.overhead-targetOverhead) > c.cfg.OverheadDeadband {
			c.overhead = targetOverhead
			changed = true
			reason = joinReasons(reason, "adjusted protection factor")
		}
	} else {
		// When disabled, ensure overhead is zero.
		if c.overhead != 0 {
			c.overhead = 0
			changed = true
		}
	}

	return PolicyDecision{
		FEC: FECPolicy{
			Enabled:        c.enabled,
			Scheme:         c.scheme,
			TargetOverhead: c.overhead,
			Reason:         reason,
			At:             eventTime(s),
		},
	}, changed
}

// shouldEnable decides if we should transition from "disabled" to "enabled".
// We only enable if loss is high enough AND we have a positive overhead budget.
func (c *FlexFEC03Controller) shouldEnable(lossRate, targetOverhead float64) bool {
	return lossRate >= c.cfg.FECEnableLossRate && targetOverhead > 0
}

// shouldDisable decides if we should transition from "enabled" to "disabled".
// We disable if loss is low enough OR we have no overhead budget (e.g., BWE cap reduced it to zero).
func (c *FlexFEC03Controller) shouldDisable(lossRate, targetOverhead float64) bool {
	return lossRate <= c.cfg.FECDisableLossRate || targetOverhead <= 0
}
