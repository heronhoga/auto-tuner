package tune

import "slices"

func findInitialPeak(samples []float32) int {
	peak := -1

	for i := 1; i < len(samples)-1; i++ {
		if samples[i] > samples[i-1] &&
			samples[i] > samples[i+1] {

			if peak == -1 || samples[i] > samples[peak] {
				peak = i
			}
		}
	}

	return peak
}

func findPeakNear(samples []float32, expected int, radius int) int {
	start := expected - radius
	stop := expected + radius

	if start < 1 {
		start = 1
	}

	if stop > len(samples)-1 {
		stop = len(samples) - 1
	}

	peak := -1

	for i := start; i < stop; i++ {
		if samples[i] > samples[i-1] &&
			samples[i] > samples[i+1] {

			if peak == -1 || samples[i] > samples[peak] {
				peak = i
			}
		}
	}

	return peak
}

func FindPitchMark(samples []float32, period float64) []int {
	if len(samples) < 3 || period <= 0 {
		return nil
	}

	first := findInitialPeak(samples)

	if first == -1 {
		return nil
	}

	marks := []int{first}
	radius := max(1, int(period*0.3))
	// The strongest peak can be anywhere in the window. Track backward too,
	// otherwise the beginning of each window may contain no synthesis grains.
	for {
		expected := float64(marks[len(marks)-1]) - period
		if expected < 1 {
			break
		}
		previous := findPeakNear(samples, int(expected), radius)
		if previous == -1 || previous >= marks[len(marks)-1] {
			break
		}
		marks = append(marks, previous)
	}
	slices.Reverse(marks)

	for {
		expected := float64(marks[len(marks)-1]) + period

		if expected >= float64(len(samples)-1) {
			break
		}

		next := findPeakNear(
			samples,
			int(expected),
			radius,
		)

		if next == -1 || next <= marks[len(marks)-1] {
			break
		}

		marks = append(marks, next)
	}

	return marks
}
