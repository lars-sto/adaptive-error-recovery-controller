package recovery

import "time"

// NetworkStats input stats (from Stats Interceptor or any adapter)
type NetworkStats struct {
	RTTMs          int     // round-trip time in milliseconds
	LossRate       float64 // 0.0..1.0 (e.g., 0.02 = 2%)
	JitterMs       int     // jitter in milliseconds
	TargetBitrate  float64
	CurrentBitrate float64
	Timestamp      time.Time // when these stats were observed
}

// StatsSource provides a stream of stats. Adapter layer can wrap Pion stats interceptor
type StatsSource interface {
	Stats() <-chan NetworkStats
}

// NACKPolicy policy for NACK Interceptor
type NACKPolicy struct {
	Enabled bool
	Reason  string
	At      time.Time
}

type FECScheme string

const (
	FECSchemeNone      FECScheme = "none"
	FECSchemeFlexFEC03 FECScheme = "flexfec03"
)

type CoverageMode string

const (
	CoverageModeInterleaved CoverageMode = "interleaved"
	// reserved for later:
	CoverageModeContiguous CoverageMode = "contiguous"
)

type FECPolicy struct {
	Enabled bool
	Scheme  FECScheme

	TargetOverhead float64

	// Apply-ready actuator knobs (what interceptor needs)
	NumMediaPackets uint32 // k (static)
	NumFECPackets   uint32 // r (dynamic)
	CoverageMode    CoverageMode

	InterleaveStride uint32
	BurstSpan        uint32

	Reason string
	At     time.Time
}

type PolicyDecision struct {
	FEC FECPolicy
}

// PolicySink receives policy decisions. Adapter layer can forward to policy pipes.
type PolicySink interface {
	Publish(decision PolicyDecision)
}

type DecisionObserver interface {
	OnSample(stats NetworkStats, decision PolicyDecision, changed bool)
}
