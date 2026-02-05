package recovery

// Config provides the policy engine config
type Config struct {
	// Selected scheme for the Engine. NewEngine() dispatches to a scheme-specific controller.
	Scheme FECScheme

	// FEC thresholds (simple v1 hysteresis)
	FECEnableLossRate  float64 // enable if loss >= this
	FECDisableLossRate float64 // disable if loss <= this

	// FEC overhead bounds
	MinOverhead      float64
	MaxOverhead      float64
	OverheadDeadband float64
}

func DefaultConfig() Config {
	return Config{
		Scheme: FECSchemeFlexFEC03,

		FECEnableLossRate:  0.03, // 3%
		FECDisableLossRate: 0.01, // 1%

		MinOverhead: 0.00,
		MaxOverhead: 0.25,

		OverheadDeadband: 0.02,
	}
}
