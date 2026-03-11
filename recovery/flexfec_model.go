package recovery

type FlexFECRecommendation struct {
	TargetOverhead float64
	Reason         string
}

// FlexFECModel produces a raw protection recommendation for the FlexFEC mechanism
type FlexFECModel interface {
	Recommend(s NetworkStats) FlexFECRecommendation
}

type TableBasedFlexFECModel struct{}

func NewTableBasedFlexFECModel() *TableBasedFlexFECModel {
	return &TableBasedFlexFECModel{}
}

func (m *TableBasedFlexFECModel) Recommend(s NetworkStats) FlexFECRecommendation {
	targetOverhead := GetLossProtFactor(s.RTTMs, s.LossRate)

	protectionFactorScale := 1.00
	targetOverhead = targetOverhead * protectionFactorScale

	return FlexFECRecommendation{
		TargetOverhead: targetOverhead,
		Reason:         "table-based FlexFEC recommendation",
	}
}
