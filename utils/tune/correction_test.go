package tune

import (
	"math"
	"sync"
	"testing"

	"github.com/heronhoga/auto-tuner/utils/yin"
)

func midiFrequency(midi float64) float64 {
	return 440 * math.Exp2((midi-69)/12)
}

func TestCorrectionRatio(t *testing.T) {
	tests := []struct {
		name       string
		key        int
		scale      Scale
		inputMIDI  float64
		strength   float64
		targetMIDI float64
	}{
		{"down to C", 0, Major, 60.4, 1, 60},
		{"up to D", 0, Major, 61.7, 1, 62},
		{"major third", 0, Major, 63.3, 1, 64},
		{"minor third", 0, Minor, 63.3, 1, 63},
		{"sharp key", 1, Major, 60.8, 1, 61},
		{"G major F sharp", 7, Major, 65.6, 1, 66},
		{"A natural minor G", 9, Minor, 67.8, 1, 67},
		{"next octave", 0, Major, 71.7, 1, 72},
		{"previous octave", 0, Major, 59.2, 1, 59},
		{"half strength", 0, Major, 63.3, 0.5, 63.65},
		{"zero strength", 0, Major, 63.3, 0, 63.3},
		{"already in tune", 0, Major, 69, 1, 69},
		{"tie prefers lower note", 0, Major, 61, 1, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := CorrectionSettings{Enabled: true, Key: tt.key, Scale: tt.scale, Strength: tt.strength}
			input := midiFrequency(tt.inputMIDI)
			actual := input * settings.Ratio(input)
			if expected := midiFrequency(tt.targetMIDI); math.Abs(actual-expected) > 1e-8 {
				t.Fatalf("corrected frequency %.6f, want %.6f", actual, expected)
			}
		})
	}
	settings := DefaultCorrectionSettings()
	for _, frequency := range []float64{0, -100, math.NaN(), math.Inf(1)} {
		if got := settings.Ratio(frequency); got != 1 {
			t.Errorf("invalid frequency %v: ratio %v, want 1", frequency, got)
		}
	}
	settings.Enabled = false
	if got := settings.Ratio(435); got != 1 {
		t.Errorf("disabled ratio %v, want 1", got)
	}
}

func processCorrection(corrector *AutoCorrector, input []float32, chunkSize int) []float32 {
	output := make([]float32, len(input))
	for start := 0; start < len(input); start += chunkSize {
		end := min(start+chunkSize, len(input))
		corrector.Process(input[start:end], output[start:end])
	}
	return output
}

func assertPitch(t *testing.T, samples []float32, target float64) {
	t.Helper()
	detector := yin.New(48000)
	var sumSquares float64
	for _, sample := range samples {
		if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
			t.Fatal("non-finite output")
		}
		sumSquares += float64(sample) * float64(sample)
	}
	if rms := math.Sqrt(sumSquares / float64(len(samples))); rms < 0.15 || rms > 0.6 {
		t.Fatalf("unexpected RMS %.4f", rms)
	}
	for start := 0; start+4096 <= len(samples); start += 4096 {
		note, err := detector.Detect(samples[start : start+4096])
		if err != nil {
			t.Fatalf("no pitch at sample %d: %v", start, err)
		}
		if cents := 1200 * math.Log2(note.Frequency/target); math.Abs(cents) > 15 {
			t.Fatalf("pitch %.2f Hz, want %.2f Hz (error %+.1f cents)", note.Frequency, target, cents)
		}
	}
}

func TestAutoCorrectorAudio(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		settings CorrectionSettings
		target   float64
	}{
		{"raise flat A", 430, DefaultCorrectionSettings(), 440},
		{"lower sharp A", 450, DefaultCorrectionSettings(), 440},
		{"major third", midiFrequency(63.3), DefaultCorrectionSettings(), midiFrequency(64)},
		{"minor third", midiFrequency(63.3), CorrectionSettings{true, 1, 0, Minor}, midiFrequency(63)},
		{"half strength", 430, CorrectionSettings{true, 0.5, 0, Major}, math.Sqrt(430 * 440)},
		{"low voice", 108, DefaultCorrectionSettings(), 110},
		{"sharp key", midiFrequency(60.8), CorrectionSettings{true, 1, 1, Major}, midiFrequency(61)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrector := NewAutoCorrector(48000)
			corrector.SetSettings(tt.settings)
			output := processCorrection(corrector, generateSine(tt.input, 48000, 48000), 257)
			assertPitch(t, output[3*psolaWindowSize:len(output)-psolaWindowSize], tt.target)
		})
	}
}

func TestAutoCorrectorBypass(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		corrector := NewAutoCorrector(48000)
		settings := corrector.Settings()
		settings.Enabled = enabled
		if enabled {
			settings.Strength = 0
		}
		corrector.SetSettings(settings)
		input := generateSine(435, 48000, 20000)
		output := processCorrection(corrector, input, 127)
		for i, sample := range output {
			var expected float32
			if i >= psolaWindowSize {
				expected = input[i-psolaWindowSize]
			}
			if math.Abs(float64(sample-expected)) > 1e-6 {
				t.Fatalf("enabled=%v: bypass differs at sample %d", enabled, i)
			}
		}
	}
}

func TestAutoCorrectorSilenceAndUnvoiced(t *testing.T) {
	input := make([]float32, 16000)
	for _, sample := range processCorrection(NewAutoCorrector(48000), input, 512) {
		if sample != 0 {
			t.Fatal("silence produced nonzero audio")
		}
	}
	// Deterministic white noise has no stable fundamental and must pass through.
	state := uint32(1)
	for i := range input {
		state = state*1664525 + 1013904223
		input[i] = float32(float64(state)/math.MaxUint32-0.5) * 0.2
	}
	output := processCorrection(NewAutoCorrector(48000), input, 512)
	for i := psolaWindowSize; i < len(output); i++ {
		if math.Abs(float64(output[i]-input[i-psolaWindowSize])) > 1e-6 {
			t.Fatalf("unvoiced audio changed at sample %d", i)
		}
	}
}

func TestAutoCorrectorChunkIndependence(t *testing.T) {
	input := generateSine(435, 48000, 24000)
	expected := processCorrection(NewAutoCorrector(48000), input, len(input))
	for _, chunkSize := range []int{1, 127, 512, 3000} {
		actual := processCorrection(NewAutoCorrector(48000), input, chunkSize)
		for i := range actual {
			if actual[i] != expected[i] {
				t.Fatalf("chunk size %d changes output at sample %d", chunkSize, i)
			}
		}
	}
}

func TestAutoCorrectorLiveSettings(t *testing.T) {
	corrector := NewAutoCorrector(48000)
	input := generateSine(434, 48000, 48000)
	output := processCorrection(corrector, input, 512)
	assertPitch(t, output[24000:], 440)

	settings := corrector.Settings()
	settings.Scale = Minor
	corrector.SetSettings(settings)
	output = processCorrection(corrector, input, 512)
	assertPitch(t, output[24000:], midiFrequency(68)) // C minor contains Ab, not A.

	settings.Enabled = false
	corrector.SetSettings(settings)
	output = processCorrection(corrector, input, 512)
	assertPitch(t, output[24000:], 434)
}

func TestAutoCorrectorConcurrentSettings(t *testing.T) {
	corrector := NewAutoCorrector(48000)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 1000; i++ {
			settings := corrector.Settings()
			settings.Key = i % 12
			settings.Strength = float64(i%101) / 100
			corrector.SetSettings(settings)
		}
	}()
	processCorrection(corrector, generateSine(435, 48000, 12000), 128)
	workers.Wait()
}

func TestAutoCorrectorVoiceAndSilenceTransitions(t *testing.T) {
	// A harmonic-rich low voice exercises long grains, restarting synthesis
	// after silence, and reuse of the overlap-add ring over many windows.
	const segmentLength = 24000
	input := make([]float32, 4*segmentLength)
	for _, start := range []int{0, 2 * segmentLength} {
		for i := 0; i < segmentLength; i++ {
			phase := 2 * math.Pi * 108 * float64(i) / 48000
			input[start+i] = float32(0.35*math.Sin(phase) + 0.1*math.Sin(2*phase) + 0.05*math.Sin(3*phase))
		}
	}
	output := processCorrection(NewAutoCorrector(48000), input, 257)
	for _, start := range []int{0, 2 * segmentLength} {
		assertPitch(t, output[start+3*psolaWindowSize:start+segmentLength], 110)
	}
	for _, start := range []int{segmentLength, 3 * segmentLength} {
		for i := start + 2*psolaWindowSize; i < start+segmentLength; i++ {
			if math.Abs(float64(output[i])) > 1e-6 {
				t.Fatalf("stale audio leaked into silence at sample %d", i)
			}
		}
	}
}
