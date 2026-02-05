package recovery

import (
	"testing"
	"time"
)

func TestClamp_InRangeReturnsSame(t *testing.T) {
	got := clamp(0.5, 0.0, 1.0)
	if got != 0.5 {
		t.Fatalf("expected 0.5, got %v", got)
	}
}

func TestClamp_BelowMinReturnsMin(t *testing.T) {
	got := clamp(-1.0, 0.0, 1.0)
	if got != 0.0 {
		t.Fatalf("expected 0.0, got %v", got)
	}
}

func TestClamp_AboveMaxReturnsMax(t *testing.T) {
	got := clamp(2.0, 0.0, 1.0)
	if got != 1.0 {
		t.Fatalf("expected 1.0, got %v", got)
	}
}

func TestJoinReasons_FirstEmptyReturnsSecond(t *testing.T) {
	got := joinReasons("", "BWE cap")
	if got != "BWE cap" {
		t.Fatalf("expected %q, got %q", "BWE cap", got)
	}
}

func TestJoinReasons_SecondEmptyReturnsFirst(t *testing.T) {
	got := joinReasons("adjusted protection factor", "")
	if got != "adjusted protection factor" {
		t.Fatalf("expected %q, got %q", "adjusted protection factor", got)
	}
}

func TestJoinReasons_BothNonEmptyJoinsWithSeparator(t *testing.T) {
	got := joinReasons("BWE cap", "adjusted protection factor")
	want := "BWE cap | adjusted protection factor"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestJoinReasons_BothEmptyReturnsEmpty(t *testing.T) {
	got := joinReasons("", "")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestEventTime_UsesStatsTimestampIfPresent(t *testing.T) {
	ts := time.Unix(12345, 0)
	s := NetworkStats{Timestamp: ts}

	got := eventTime(s)
	if !got.Equal(ts) {
		t.Fatalf("expected %v, got %v", ts, got)
	}
}

func TestEventTime_WhenTimestampZero_ReturnsNonZeroTime(t *testing.T) {
	s := NetworkStats{} // zero timestamp

	got := eventTime(s)
	if got.IsZero() {
		t.Fatalf("expected non-zero time")
	}
}
