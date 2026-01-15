package recovery

//
// Minimal helpers for local prototyping (optional):
// - ChanSource: feed stats via channel
// - ChanSink: observe decisions via channel
//

type ChanSource struct{ ch <-chan NetworkStats }

func NewChanSource(ch <-chan NetworkStats) *ChanSource { return &ChanSource{ch: ch} }
func (s *ChanSource) Stats() <-chan NetworkStats       { return s.ch }

type ChanSink struct{ ch chan PolicyDecision }

func NewChanSink(buf int) *ChanSink {
	if buf <= 0 {
		buf = 16
	}
	return &ChanSink{ch: make(chan PolicyDecision, buf)}
}
func (s *ChanSink) Publish(d PolicyDecision)         { s.ch <- d }
func (s *ChanSink) Decisions() <-chan PolicyDecision { return s.ch }
