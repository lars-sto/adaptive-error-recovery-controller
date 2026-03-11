package recovery

// AsFlexFECPolicy extracts a FlexFEC policy from a policy decision
func AsFlexFECPolicy(d PolicyDecision) (FlexFECPolicy, bool) {
	if d.Mechanism != MechanismFlexFEC03 {
		return FlexFECPolicy{}, false
	}

	p, ok := d.Policy.(FlexFECPolicy)
	if !ok {
		return FlexFECPolicy{}, false
	}
	return p, true
}

func NewFlexFECDecision(p FlexFECPolicy) PolicyDecision {
	return PolicyDecision{
		Mechanism: MechanismFlexFEC03,
		Policy:    p,
	}
}
