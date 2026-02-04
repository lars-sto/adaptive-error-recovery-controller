package recovery

import (
	"math"
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

	// 1) Compute baseline protection overhead from RTT + loss (interpolated table).
	targetOverhead := GetLossProtFactor(s.RTTMs, s.LossRate)

	// 2) Clamp to feasible bounds from config.
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
		if s.LossRate >= c.cfg.FECEnableLossRate && targetOverhead > 0 {
			newEnabled = true
		}
	} else {
		if s.LossRate <= c.cfg.FECDisableLossRate || targetOverhead <= 0 {
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
		if math.Abs(c.overhead-targetOverhead) > 0.02 {
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
