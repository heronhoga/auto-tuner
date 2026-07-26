package yin

func absoluteThreshold(cmnd []float64, threshold float64) int {
	for tau := 2; tau < len(cmnd); tau++ {
		if cmnd[tau] < threshold {
			
			for tau+1 < len(cmnd) && cmnd[tau+1] < cmnd[tau] {
				tau++
			}

			return tau
		}
	}

	return -1
}