package recovery

func newMechanismControllers(cfg Config) []MechanismController {
	var controllers []MechanismController

	if cfg.FlexFEC != nil {
		controllers = append(controllers, NewFlexFECController(*cfg.FlexFEC, nil))
	}

	return controllers
}
