package yin

import (
	"errors"
	"math"
)

type Detector struct {
	SampleRate float64
	Threshold  float64
}

func New(sampleRate float64) *Detector {
	return &Detector{
		SampleRate: sampleRate,
		Threshold:  0.10,
	}
}

func (d *Detector) Detect(samples []float32) (NoteResult, error) {
	diff := difference(samples)
	cmnd := cumulativeMeanNormalizedDifference(diff)

	tau := absoluteThreshold(cmnd, d.Threshold)
	if tau == -1 {
		return NoteResult{}, errors.New("No Pitch Detected")
	}

	tauHat := parabolicInterpolation(cmnd, tau)
	frequency := d.SampleRate / tauHat

	if frequency <= 0 || math.IsNaN(frequency) || math.IsInf(frequency, 0) {
		return NoteResult{}, errors.New("No Pitch Detected")
	}

	return FrequencyToNote(frequency), nil
}