package recovery

// Config provides the policy engine config
type Config struct {
	FlexFEC *FlexFECConfig
	NACK    *NACKConfig
	RED     *REDConfig
}

type FlexFECConfig struct {
	EnableLossRate   float64
	DisableLossRate  float64
	MinOverhead      float64
	MaxOverhead      float64
	OverheadDeadband float64

	NumMediaPackets  uint32
	CoverageMode     CoverageMode
	InterleaveStride uint32
	BurstSpan        uint32
}

type NACKConfig struct{}

type REDConfig struct{}

func DefaultConfig() Config {
	const k = uint32(10)

	return Config{
		FlexFEC: &FlexFECConfig{
			EnableLossRate:   0.03,
			DisableLossRate:  0.01,
			MinOverhead:      0.00,
			MaxOverhead:      0.25,
			OverheadDeadband: 0.02,
			NumMediaPackets:  k,
			CoverageMode:     CoverageModeInterleaved,
			InterleaveStride: 1,
			BurstSpan:        k,
		},
	}
}
