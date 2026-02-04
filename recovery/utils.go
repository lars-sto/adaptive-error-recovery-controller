package recovery

import "time"

// eventTime uses the stats timestamp if available; otherwise falls back to wall clock.
func eventTime(s NetworkStats) time.Time {
	if !s.Timestamp.IsZero() {
		return s.Timestamp
	}
	return time.Now()
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func joinReasons(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + " | " + b
}
