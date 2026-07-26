package yin

import (
	"errors"
	"fmt"
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

func (d *Detector) Detect(samples []float32) (float64, error) {
	diff := difference(samples)
	cmnd := cumulativeMeanNormalizedDifference(diff)

	tau := absoluteThreshold(cmnd, d.Threshold)
	if tau == -1 {
		return 0, errors.New("No Pitch Detected")
	}

	println("tau: ", tau)

	tauHat := parabolicInterpolation(cmnd, tau)
	println("parabolic: ", tauHat)

	frequency := d.SampleRate / tauHat
	println("frequency: ", frequency)

	for i := 100; i <= 115; i++ {
		fmt.Printf("%3d %.8f\n", i, cmnd[i])
	}

	return frequency, nil

}