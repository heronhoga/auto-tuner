package ui

import (
	"sync"

	utils "github.com/heronhoga/auto-tuner/utils/audio"
)

type Waveform struct {
	mu sync.RWMutex
	Samples []float32
}

func NewWaveform() *Waveform {
	return &Waveform{}
}

func (w *Waveform) Update(frame utils.Frame) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Samples = append([]float32(nil), frame.Samples... )
}

func (w *Waveform) GetSamples() []float32 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return append([]float32(nil), w.Samples...)
}