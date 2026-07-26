package yin

func difference(samples []float32) []float64 {
	maxLag := len(samples)/2

	diff := make([]float64, maxLag)

	for tau := 0; tau < maxLag; tau++ {
		var sum float64

		for j := 0; j < maxLag; j++ {
			delta := float64(samples[j] - samples[j+tau])
			sum += delta * delta
		}
		diff[tau] = sum
	}

	return diff
}