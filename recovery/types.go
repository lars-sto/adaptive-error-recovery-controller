package recovery

import "time"

type StreamKey struct {
	MediaSSRC uint32
}

// NetworkStats input stats (from Stats Interceptor or any adapter)
type NetworkStats struct {
	Stream StreamKey

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

type MechanismKind string

const (
	MechanismNone      MechanismKind = "none"
	MechanismFlexFEC03 MechanismKind = "flexfec03"
)

type CoverageMode string

const (
	CoverageModeInterleaved CoverageMode = "interleaved"
	// reserved for later:
	CoverageModeContiguous CoverageMode = "contiguous"
)

type FlexFECPolicy struct {
	Enabled   bool
	Mechanism MechanismKind

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
	Stream    StreamKey
	Mechanism MechanismKind
	Policy    any
}

// PolicySink receives policy decisions. Adapter layer can forward to policy pipes
type PolicySink interface {
	Publish(decision PolicyDecision)
}

// DecisionObserver OnSample may be called multiple times for a single stats sample
// when multiple mechanism controllers are registered
type DecisionObserver interface {
	OnSample(stats NetworkStats, decision PolicyDecision, changed bool)
}
