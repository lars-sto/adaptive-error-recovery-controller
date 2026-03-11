package recovery

// MechanismController evaluates runtime input and returns mechanism-specific policy decision
type MechanismController interface {
	Decide(s NetworkStats) (PolicyDecision, bool)
}
