package recovery

// Config provides the policy engine config
type Config struct {
	// FEC thresholds (simple v1 heuristic)
	FECEnableLossRate  float64 // enable FEC if loss >= this
	FECDisableLossRate float64 // disable FEC if loss <= this

	// FEC overhead bounds
	MinOverhead float64
	MaxOverhead float64
}

func DefaultConfig() Config {
	return Config{
		FECEnableLossRate:  0.03, // 3%
		FECDisableLossRate: 0.01, // 1%

		MinOverhead: 0.00,
		MaxOverhead: 0.25,
	}
}
