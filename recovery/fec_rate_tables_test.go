package recovery

import "testing"

func TestGetLossProtFactor_MonotonicInLoss(t *testing.T) {
	// Test representative RTT zones:
	// - low RTT (NACK-friendly)
	// - mid RTT (hybrid)
	// - high RTT (FEC-dominant)
	rtts := []int{
		20,  // < 60 ms
		100, // 60–150 ms
		300, // > 150 ms
	}

	// Increasing loss samples (covering table coordinates and in-betweens)
	losses := []float64{
		0.00,
		0.01,
		0.03,
		0.05,
		0.10,
		0.20,
		0.50,
	}

	for _, rtt := range rtts {
		var prev float64 = -1.0

		for _, loss := range losses {
			val := GetLossProtFactor(rtt, loss)

			if prev >= 0 && val < prev {
				t.Fatalf(
					"protection factor not monotonic for RTT=%dms: loss %.3f -> %.3f decreased from %.3f to %.3f",
					rtt, loss, loss, prev, val,
				)
			}

			prev = val
		}
	}
}

func TestGetLossProtFactor_WithinExpectedRange(t *testing.T) {
	rtts := []int{10, 80, 200, 500}
	losses := []float64{0.0, 0.01, 0.05, 0.10, 0.30, 0.50}

	for _, rtt := range rtts {
		for _, loss := range losses {
			val := GetLossProtFactor(rtt, loss)
			if val < 0.0 || val > 1.0 {
				t.Fatalf(
					"protection factor out of range for RTT=%dms loss=%.3f: got %.3f",
					rtt, loss, val,
				)
			}
		}
	}
}

func TestGetLossProtFactor_OutOfRangeInputsDoNotPanic(t *testing.T) {
	cases := []struct {
		rtt  int
		loss float64
	}{
		{-50, -0.1},
		{0, -1.0},
		{10, 2.0},
		{1000, 10.0},
	}

	for _, c := range cases {
		val := GetLossProtFactor(c.rtt, c.loss)
		if val < 0.0 || val > 1.0 {
			t.Fatalf(
				"unexpected value for out-of-range input RTT=%d loss=%.3f: got %.3f",
				c.rtt, c.loss, val,
			)
		}
	}
}

func TestGetLossProtFactor_Deterministic(t *testing.T) {
	rtt := 120
	loss := 0.07

	v1 := GetLossProtFactor(rtt, loss)
	v2 := GetLossProtFactor(rtt, loss)
	v3 := GetLossProtFactor(rtt, loss)

	if v1 != v2 || v2 != v3 {
		t.Fatalf(
			"non-deterministic output for same input: %.6f %.6f %.6f",
			v1, v2, v3,
		)
	}
}
