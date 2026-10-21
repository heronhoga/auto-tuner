package tune

import (
	"math"
	"sync"
)

type Scale string

const (
	Major Scale = "Major"
	Minor Scale = "Minor"
)

type CorrectionSettings struct {
	Enabled  bool
	Strength float64 // 0 = original pitch, 1 = fully corrected.
	Key      int     // Pitch class: C = 0, C# = 1, ..., B = 11.
	Scale    Scale   // Minor means natural minor.
}

func DefaultCorrectionSettings() CorrectionSettings {
	return CorrectionSettings{Enabled: true, Strength: 1, Key: 0, Scale: Major}
}

// Ratio moves toward the nearest scale note in logarithmic pitch (semitones),
// rather than mixing dry and shifted audio, which would produce two pitches.
func (s CorrectionSettings) Ratio(frequency float64) float64 {
	if !s.Enabled || s.Strength <= 0 || math.IsNaN(s.Strength) ||
		frequency <= 0 || math.IsNaN(frequency) || math.IsInf(frequency, 0) {
		return 1
	}
	intervals := [...]int{0, 2, 4, 5, 7, 9, 11}
	if s.Scale == Minor {
		intervals = [...]int{0, 2, 3, 5, 7, 8, 10}
	}
	midi := 69 + 12*math.Log2(frequency/440)
	key := (s.Key%12 + 12) % 12
	nearest, distance := 0, math.Inf(1)
	// Search neighboring octaves, including notes across the B/C boundary.
	for candidate := int(math.Floor(midi)) - 12; candidate <= int(math.Ceil(midi))+12; candidate++ {
		pitchClass := ((candidate-key)%12 + 12) % 12
		for _, interval := range intervals {
			if pitchClass == interval {
				if delta := math.Abs(float64(candidate) - midi); delta < distance {
					nearest, distance = candidate, delta
				}
				break
			}
		}
	}
	return math.Exp2((float64(nearest) - midi) * math.Min(s.Strength, 1) / 12)
}

// AutoCorrector is processed by one audio worker. Settings may be changed by
// the UI concurrently; each analysis window uses a complete settings snapshot.
type AutoCorrector struct {
	mu       sync.RWMutex
	settings CorrectionSettings
	shifter  *PSOLAProcessor
}

func NewAutoCorrector(sampleRate float64) *AutoCorrector {
	c := &AutoCorrector{
		settings: DefaultCorrectionSettings(),
		shifter:  NewPSOLAProcessor(sampleRate, 1),
	}
	c.shifter.ratioForFrequency = func(frequency float64) float64 {
		return c.Settings().Ratio(frequency)
	}
	return c
}

func (c *AutoCorrector) Settings() CorrectionSettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.settings
}

func (c *AutoCorrector) SetSettings(settings CorrectionSettings) {
	settings.Key = (settings.Key%12 + 12) % 12
	if settings.Scale != Minor {
		settings.Scale = Major
	}
	if math.IsNaN(settings.Strength) {
		settings.Strength = 0
	}
	settings.Strength = math.Max(0, math.Min(1, settings.Strength))
	c.mu.Lock()
	c.settings = settings
	c.mu.Unlock()
}

// Process produces one output sample per input sample, with a fixed analysis
// delay even when bypassed so toggling correction does not jump the timeline.
func (c *AutoCorrector) Process(input, output []float32) {
	c.shifter.ProcessSamples(input, output)
}
