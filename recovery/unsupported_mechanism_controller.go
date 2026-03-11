package recovery

type UnsupportedMechanismController struct {
	cfg Config
}

func NewUnsupportedMechanismController(cfg Config) *UnsupportedMechanismController {
	return &UnsupportedMechanismController{cfg: cfg}
}

func (u *UnsupportedMechanismController) Decide(s NetworkStats) (PolicyDecision, bool) {
	return PolicyDecision{
		Mechanism: MechanismNone,
		Policy:    nil,
	}, false
}
