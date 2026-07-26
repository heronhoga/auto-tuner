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

func TestDetectPitch440Hz(t *testing.T) {
	const (
		sampleRate = 48000.0
		frequency   = 440.0
		nSamples    = 4096
		threshold   = 0.10
	)

	samples := generateSine(frequency, sampleRate, nSamples)

	diff := difference(samples)
	cmnd := cumulativeMeanNormalizedDifference(diff)

	tau := absoluteThreshold(cmnd, threshold)
	if tau == -1 {
		t.Fatal("no pitch detected")
	}

	tauHat := parabolicInterpolation(cmnd, tau)
	freq := sampleRate / tauHat

	t.Logf("tau = %d", tau)
	t.Logf("tauHat = %.6f", tauHat)
	t.Logf("freq = %.6f", freq)

	expectedTau := sampleRate / frequency

	if math.Abs(float64(tau)-expectedTau) > 5 {
		t.Fatalf("expected tau around %.2f, got %d", expectedTau, tau)
	}

	if math.Abs(freq-frequency) > 1 {
		t.Fatalf("expected freq around %.2f, got %.6f", frequency, freq)
	}
}