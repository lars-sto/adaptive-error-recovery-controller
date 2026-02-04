package recovery

// UnsupportedFECController is a placeholder for future schemes.
// v1 behavior: never emits a policy change and always reports "FEC disabled".
type UnsupportedFECController struct {
	cfg Config
}

func NewUnsupportedFECController(cfg Config) *UnsupportedFECController {
	return &UnsupportedFECController{cfg: cfg}
}

func (u *UnsupportedFECController) Decide(s NetworkStats) (PolicyDecision, bool) {
	// Intentionally do not compute anything here.
	return PolicyDecision{
		FEC: FECPolicy{
			Enabled:        false,
			Scheme:         FECSchemeNone,
			TargetOverhead: 0,
			Reason:         "",
			At:             eventTime(s),
		},
	}, false
}
