package effects

import (
	"math"
	"sync"
)

// Settings controls effect amounts from 0 (off) to 1 (maximum). At maximum,
// each stage blends equal parts dry and wet to retain the direct vocal.
type Settings struct {
	Reverb float64
	Chorus float64
}

// Processor applies chorus followed by reverb. One audio worker owns Process;
// the UI may read or change settings concurrently. Delay buffers are allocated
// at construction, and Process also supports input and output sharing a slice.
type Processor struct {
	mu       sync.RWMutex
	settings Settings
	chorus   chorus
	reverb   reverb

	chorusAmount float64
	reverbAmount float64
	rampStep     float64
}

func NewProcessor(sampleRate float64) *Processor {
	if sampleRate <= 0 || math.IsNaN(sampleRate) || math.IsInf(sampleRate, 0) {
		panic("effects: sample rate must be finite and positive")
	}
	return &Processor{
		chorus:   newChorus(sampleRate),
		reverb:   newReverb(sampleRate),
		rampStep: 1 / (0.020 * sampleRate), // Smooth a full-range change over 20 ms.
	}
}

func (p *Processor) Settings() Settings {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.settings
}

func (p *Processor) SetSettings(settings Settings) {
	settings.Reverb = clampAmount(settings.Reverb)
	settings.Chorus = clampAmount(settings.Chorus)
	p.mu.Lock()
	p.settings = settings
	p.mu.Unlock()
}

func clampAmount(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	return math.Max(0, math.Min(1, value))
}

func approach(current, target, step float64) float64 {
	if math.Abs(target-current) <= step {
		return target
	}
	return current + math.Copysign(step, target-current)
}

// Process requires output to hold at least len(input) samples. The dry signal
// has no added latency. At zero amount, output is exactly dry once the short
// control ramp settles. Delay lines keep running so changes need no reset.
func (p *Processor) Process(input, output []float32) {
	settings := p.Settings()
	for i, sample := range input {
		p.chorusAmount = approach(p.chorusAmount, settings.Chorus, p.rampStep)
		p.reverbAmount = approach(p.reverbAmount, settings.Reverb, p.rampStep)

		dry := float64(sample)
		chorused := dry + 0.5*p.chorusAmount*(p.chorus.process(dry)-dry)
		wet := p.reverb.process(chorused)
		output[i] = float32(chorused + 0.5*p.reverbAmount*(wet-chorused))
	}
}
