package yin

import (
	"math"
	"testing"
)

func generateSine(freq float64, sampleRate float64, n int) []float32 {
	samples := make([]float32, n)

	for i := range samples {
		t := float64(i) / sampleRate
		samples[i] = float32(math.Sin(2 * math.Pi * freq * t))
	}

	return samples
}

func TestYIN440Hz(t *testing.T) {
	const (
		sampleRate = 48000.0
		frequency = 440.0
		nSamples = 4096
		threshold = 0.1
	)

	samples := generateSine(frequency, sampleRate, nSamples)

	diff := difference(samples)
	cmnd := cumulativeMeanNormalizedDifference(diff)

	for i := 0; i < 20; i++ {
    	t.Logf("%2d  diff=%12.3f  cmnd=%8.5f", i, diff[i], cmnd[i])
	}

	lag := absoluteThreshold(cmnd, threshold)

	expected := int(math.Round(sampleRate/frequency))

	tolerance := 5

	if math.Abs(float64(lag-expected)) > float64(tolerance) {
		t.Fatalf("expected lag near %d, got %d", expected, lag)
	}

	if lag == -1 {
    t.Fatal("no pitch detected")
	}

	if cmnd[0] != 1 {
		t.Fatal("cmnd[0] should equal 1")
	}

	if lag <= 2 {
		t.Fatalf("invalid lag: %d", lag)
	}
}