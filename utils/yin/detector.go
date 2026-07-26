package yin

import (
	"errors"
	"math"
)

type Detector struct {
	SampleRate float64
	Threshold float64
}

func New(sampleRate float64) *Detector {
	return &Detector{
		SampleRate: sampleRate,
		Threshold: 0.10,
	}
}

func (d *Detector) Detect(samples []float32) (NoteResult, error) {
	diff := difference(samples)
	cmnd := cumulativeMeanNormalizedDifference(diff)

	tau := absoluteThreshold(cmnd, d.Threshold)
	if tau == -1 {
		return NoteResult{}, errors.New("No Pitch Detected")
	}

	println("tau: ", tau)

	tauHat := parabolicInterpolation(cmnd, tau)
	println("parabolic: ", tauHat)

	frequency := d.SampleRate / tauHat
	println("frequency: ", frequency)

	if frequency <= 0 || math.IsNaN(frequency) || math.IsInf(frequency, 0) {
		return NoteResult{}, errors.New("No Pitch Detected")
	}

	return FrequencyToNote(frequency), nil

}