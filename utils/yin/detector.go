package yin

type Detector struct {
	SampleRate float64
}

func New(sampleRate float64) *Detector {
	return &Detector{
		SampleRate: sampleRate,
	}
}

// func (d *Detector) Detect(samples []float32) (float64, error)