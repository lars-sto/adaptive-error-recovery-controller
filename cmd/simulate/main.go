package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lars-sto/adaptive-error-recovery-controller/recovery"
)

func main() {
	scenario := flag.String("scenario", "loss_increasing", "scenario name (loss_increasing|loss_threshold|bwe_bottleneck)")
	out := flag.String("out", "", "output csv path (default based on scenario)")
	flag.Parse()

	outPath := *out
	if outPath == "" {
		outPath = fmt.Sprintf("simdata/%s.csv", *scenario)
	}

	cfg := recovery.DefaultConfig()
	controller := recovery.NewFlexFEC03Controller(cfg)

	obs, err := NewCSVObserver(outPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = obs.Close() }()

	start := time.Now()

	var series []recovery.NetworkStats
	switch *scenario {
	case "loss_increasing":
		series = scenario01IncreasingLoss(start)
	case "loss_threshold":
		series = scenario02LossAroundEnable(start)
	case "bwe_bottleneck":
		series = scenario03BWEBottleneck(start)
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario: %s\n", *scenario)
		os.Exit(2)
	}

	for _, s := range series {
		dec, changed := controller.Decide(s)
		if err := obs.OnSample(s, dec, changed); err != nil {
			panic(err)
		}
	}
}
