package yin

func parabolicInterpolation(cmnd []float64, tau int) float64 {
	if tau <= 0 || tau >= len(cmnd) - 1 {
		return float64(tau)
	}

	denominator := 2 * (2 *(cmnd[tau]) - cmnd[tau+1] - cmnd[tau-1])
	if denominator == 0 {
		return float64(tau)
	}


	numerator := cmnd[tau+1] - cmnd[tau-1]

	offset := numerator / denominator

	return float64(tau) + offset
}