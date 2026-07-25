package yin

func cumulativeMeanNormalizedDifference(difference []float64) []float64 {
	cmnd := make([]float64, len(difference))
	cmnd[0] = 1

	var runningSum float64

	for tau := 1; tau < len(difference); tau++ {
		runningSum += difference[tau]

		if runningSum == 0 {
			cmnd[tau] = 1
			continue
		}

		cmnd[tau] = difference[tau] * float64(tau) / runningSum
	}

	return cmnd
}