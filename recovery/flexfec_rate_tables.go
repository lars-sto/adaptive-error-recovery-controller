package recovery

// RTT thresholds in ms
var rttCoords = []float64{60, 150, 400}

// Protection-Matrix inspired from libwebrtc
// Rows: RTT zones (Low, Mid, High)
// Cols: Loss-Level (0%, 5%, 10%, 20%, 50%)
var protectionMatrix = [][]float64{
	{0.00, 0.05, 0.10, 0.20, 0.40}, // RTT < 60ms (NACK preferred)
	{0.05, 0.15, 0.25, 0.40, 0.60}, // RTT 60-150ms (Hybrid)
	{0.10, 0.25, 0.40, 0.60, 0.80}, // RTT > 150ms (FEC dominant)
}

var lossCoords = []float64{0.00, 0.05, 0.10, 0.20, 0.50}

func GetLossProtFactor(rttMs int, lossRate float64) float64 {
	rtt := float64(rttMs)

	// 1. RTT-Neighbors
	rIdx1, rIdx2, rWeight := getInterpolationParams(rtt, rttCoords)

	// 2. protection factor for RTT zone
	val1 := interpolateLoss(lossRate, protectionMatrix[rIdx1])
	val2 := interpolateLoss(lossRate, protectionMatrix[rIdx2])

	// 3. final value based on weight of RTT zones
	return val1*(1-rWeight) + val2*rWeight
}

// Interpolates loss value in RTT col
func interpolateLoss(loss float64, tableRow []float64) float64 {
	idx1, idx2, weight := getInterpolationParams(loss, lossCoords)
	return tableRow[idx1]*(1-weight) + tableRow[idx2]*weight
}

// calculates index und weight (0.0 to 1.0) for interpolation
func getInterpolationParams(val float64, coords []float64) (int, int, float64) {
	if val <= coords[0] {
		return 0, 0, 0.0
	}
	if val >= coords[len(coords)-1] {
		return len(coords) - 1, len(coords) - 1, 0.0
	}

	for i := 0; i < len(coords)-1; i++ {
		if val >= coords[i] && val <= coords[i+1] {
			weight := (val - coords[i]) / (coords[i+1] - coords[i])
			return i, i + 1, weight
		}
	}
	return 0, 0, 0.0
}
