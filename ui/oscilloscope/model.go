package oscilloscope

import "sync"

type Model struct {
	mu sync.RWMutex
	samples []float32
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) Update(samples []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.samples = append([]float32(nil), samples...)
}

func (m *Model) Samples() []float32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]float32(nil), m.samples...)
}