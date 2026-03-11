package recovery

import (
	"context"
)

// RecoveryEngine Recovery Layer component
// Orchestrates recovery logic. It receives input runtime input,
// invokes mechanism-specific controllers, and emits recovery decisions
type RecoveryEngine struct {
	cfg  Config
	src  StatsSource
	sink PolicySink

	obs DecisionObserver

	controllers []MechanismController
}

func NewRecoveryEngine(cfg Config, src StatsSource, sink PolicySink, obs DecisionObserver) *RecoveryEngine {
	return &RecoveryEngine{
		cfg:         cfg,
		src:         src,
		sink:        sink,
		obs:         obs,
		controllers: newMechanismControllers(cfg),
	}
}

func (e *RecoveryEngine) Run(ctx context.Context) {
	statsCh := e.src.Stats()

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-statsCh:
			if !ok {
				return
			}
			for _, ctrl := range e.controllers {
				decision, changed := ctrl.Decide(s)

				if e.obs != nil {
					e.obs.OnSample(s, decision, changed)
				}

				if changed {
					e.sink.Publish(decision)
				}
			}
		}
	}
}
