package recovery

// FECController decides a policy decision from a single stats sample
// It may keep internal state (hysteresis, deadbands, smoothing)
type FECController interface {
	Decide(s NetworkStats) (PolicyDecision, bool)
}
