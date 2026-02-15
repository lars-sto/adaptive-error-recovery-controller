package recovery

// Config provides the policy engine config
type Config struct {
	Scheme FECScheme

	// Hysteresis
	FECEnableLossRate  float64
	FECDisableLossRate float64

	// Overhead bounds
	MinOverhead      float64
	MaxOverhead      float64
	OverheadDeadband float64

	// Actuator static defaults (v0)
	NumMediaPackets  uint32 // k
	CoverageMode     CoverageMode
	InterleaveStride uint32
	BurstSpan        uint32
}

func DefaultConfig() Config {
	const k = uint32(10)

	return Config{
		Scheme: FECSchemeFlexFEC03,

		FECEnableLossRate:  0.03,
		FECDisableLossRate: 0.01,

		MinOverhead:      0.00,
		MaxOverhead:      0.25,
		OverheadDeadband: 0.02,

		NumMediaPackets:  k,
		CoverageMode:     CoverageModeInterleaved,
		InterleaveStride: 1,
		BurstSpan:        k,
	}
}
