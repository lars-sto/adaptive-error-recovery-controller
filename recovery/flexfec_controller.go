package recovery

import (
	"math"
)

type FlexFECController struct {
	cfg   FlexFECConfig
	model FlexFECModel

	enabled  bool
	scheme   MechanismKind
	overhead float64
}

func NewFlexFECController(cfg FlexFECConfig, model FlexFECModel) *FlexFECController {
	if model == nil {
		model = NewTableBasedFlexFECModel()
	}

	return &FlexFECController{
		cfg:      cfg,
		model:    model,
		enabled:  false,
		scheme:   MechanismFlexFEC03,
		overhead: 0.0,
	}
}

func (c *FlexFECController) Decide(s NetworkStats) (PolicyDecision, bool) {
	changed := false

	rec := c.model.Recommend(s)
	targetOverhead := rec.TargetOverhead
	reason := rec.Reason

	// 1) Clamp to feasible bounds from config
	targetOverhead = clamp(targetOverhead, c.cfg.MinOverhead, c.cfg.MaxOverhead)

	// 2) Bandwidth-awareness (BWE veto):
	// If FEC would exceed the target bitrate, cap overhead to fit the budget
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

	// 3) Enable/disable hysteresis based on loss thresholds
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

	// 4) Overhead update with deadband (avoid updates on tiny fluctuations)
	if c.enabled {
		if math.Abs(c.overhead-targetOverhead) > c.cfg.OverheadDeadband {
			c.overhead = targetOverhead
			changed = true
			reason = joinReasons(reason, "adjusted protection factor")
		}
	} else {
		// When disabled, ensure overhead is zero
		if c.overhead != 0 {
			c.overhead = 0
			changed = true
		}
	}

	// 5) derive (k,r) for actuator
	k := c.cfg.NumMediaPackets
	var r uint32
	if c.enabled && k > 0 {
		rf := math.Round(float64(k) * c.overhead)
		if rf < 0 {
			rf = 0
		}
		if rf > float64(k) {
			rf = float64(k)
		}
		r = uint32(rf)
	}

	// if r==0 then effective disable
	if c.enabled && r == 0 {
		// keep enabled=false semantics stable for actuator
		c.enabled = false
		if c.overhead != 0 {
			c.overhead = 0
		}
		changed = true
		reason = joinReasons(reason, "FEC disabled: no overhead budget")
	}

	return NewFlexFECDecision(s.Stream, FlexFECPolicy{
		Enabled:        c.enabled,
		Mechanism:      c.scheme,
		TargetOverhead: c.overhead,

		NumMediaPackets:  k,
		NumFECPackets:    r,
		CoverageMode:     c.cfg.CoverageMode,
		InterleaveStride: c.cfg.InterleaveStride,
		BurstSpan:        c.cfg.BurstSpan,

		Reason: reason,
		At:     eventTime(s),
	}), changed
}

func (c *FlexFECController) shouldEnable(lossRate, targetOverhead float64) bool {
	return lossRate >= c.cfg.EnableLossRate && targetOverhead > 0
}

func (c *FlexFECController) shouldDisable(lossRate, targetOverhead float64) bool {
	return lossRate <= c.cfg.DisableLossRate || targetOverhead <= 0
}
