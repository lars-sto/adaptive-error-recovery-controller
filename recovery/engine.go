package recovery

import (
	"context"
)

// Engine calculates FEC policies based on network stats
type Engine struct {
	cfg  Config
	src  StatsSource
	sink PolicySink

	// internal state (kept minimal for v1)
	ctrl FECController
}

func NewEngine(cfg Config, src StatsSource, sink PolicySink) *Engine {
	var ctrl FECController

	// Scheme dispatch happens once at construction time.
	// Run() remains generic and does not contain scheme-specific logic.
	switch cfg.Scheme {
	case FECSchemeFlexFEC03:
		ctrl = NewFlexFEC03Controller(cfg)
	default:
		ctrl = NewUnsupportedFECController(cfg)
	}

	return &Engine{
		cfg:  cfg,
		src:  src,
		sink: sink,
		ctrl: ctrl,
	}
}

// Run consumes stats and emits policy decisions.
// v1 behavior: only emits when a decision changes, and uses the stats timestamp.
func (e *Engine) Run(ctx context.Context) {
	statsCh := e.src.Stats()

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-statsCh:
			if !ok {
				return
			}
			decision, changed := e.ctrl.Decide(s)
			if changed {
				e.sink.Publish(decision)
			}
		}
	}
}
