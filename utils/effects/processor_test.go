package effects

import (
	"math"
	"sync"
	"testing"
)

const testSampleRate = 48000

func testSignal(length int) []float32 {
	input := make([]float32, length)
	for i := range input {
		phase := 2 * math.Pi * float64(i) / testSampleRate
		input[i] = float32(0.4*math.Sin(220*phase) + 0.2*math.Sin(437*phase))
	}
	return input
}

func processChunks(p *Processor, input []float32, chunkSize int) []float32 {
	output := make([]float32, len(input))
	for start := 0; start < len(input); start += chunkSize {
		end := min(start+chunkSize, len(input))
		p.Process(input[start:end], output[start:end])
	}
	return output
}

func readyProcessor(settings Settings) *Processor {
	p := NewProcessor(testSampleRate)
	p.SetSettings(settings)
	// Settle control ramps using silence before measuring an impulse response.
	silence := make([]float32, 2048)
	p.Process(silence, silence)
	return p
}

func energy(samples []float32) float64 {
	var result float64
	for _, sample := range samples {
		result += float64(sample) * float64(sample)
	}
	return result
}

func assertFiniteAndBounded(t *testing.T, samples []float32) {
	t.Helper()
	for i, sample := range samples {
		if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) || math.Abs(float64(sample)) > 1 {
			t.Fatalf("invalid output %v at sample %d", sample, i)
		}
	}
}

func TestEffectsBypass(t *testing.T) {
	input := testSignal(24000)
	input[0] = 1.25 // Bypass must not even limit an over-range dry sample.
	output := processChunks(NewProcessor(testSampleRate), input, 257)
	for i := range input {
		if output[i] != input[i] {
			t.Fatalf("zero-amount bypass changed sample %d: %v != %v", i, output[i], input[i])
		}
	}
}

func TestEffectsSilence(t *testing.T) {
	for _, settings := range []Settings{{Chorus: 1}, {Reverb: 1}, {Chorus: 1, Reverb: 1}} {
		p := readyProcessor(settings)
		for _, sample := range processChunks(p, make([]float32, 24000), 512) {
			if sample != 0 {
				t.Fatalf("settings %+v produced audio from silence", settings)
			}
		}
	}
}

func TestChorusDelayedVoices(t *testing.T) {
	input := make([]float32, 4800)
	input[0] = 0.5
	output := processChunks(readyProcessor(Settings{Chorus: 1}), input, 127)
	assertFiniteAndBounded(t, output)
	if output[0] <= 0 {
		t.Fatal("chorus removed the direct signal")
	}
	// A chorus must add delayed voices, rather than just change dry volume.
	if energy(output[480:1920]) < 1e-4 {
		t.Fatal("no delayed chorus voices between 10 and 40 ms")
	}
	if energy(output[2400:]) != 0 {
		t.Fatal("chorus unexpectedly feeds back beyond its short delays")
	}
}

func TestReverbDecayingTail(t *testing.T) {
	input := make([]float32, 3*testSampleRate)
	input[0] = 0.5
	output := processChunks(readyProcessor(Settings{Reverb: 1}), input, 257)
	assertFiniteAndBounded(t, output)
	if output[0] <= 0 {
		t.Fatal("reverb removed the direct signal")
	}
	earlyTail := energy(output[testSampleRate/5 : testSampleRate*7/10])
	lateTail := energy(output[2*testSampleRate : testSampleRate*5/2])
	if earlyTail < 1e-7 {
		t.Fatal("reverb has no sustained tail after 200 ms")
	}
	if lateTail >= earlyTail*0.01 {
		t.Fatalf("reverb did not decay: early energy %g, late energy %g", earlyTail, lateTail)
	}
}

func TestIncreasingAmountsIncreaseWetSignal(t *testing.T) {
	input := make([]float32, testSampleRate)
	input[0] = 0.5
	for _, effect := range []string{"chorus", "reverb"} {
		t.Run(effect, func(t *testing.T) {
			var previous float64
			for _, amount := range []float64{0.25, 0.5, 1} {
				settings := Settings{Chorus: amount}
				if effect == "reverb" {
					settings = Settings{Reverb: amount}
				}
				output := processChunks(readyProcessor(settings), input, 512)
				wetEnergy := energy(output[1:])
				if wetEnergy <= previous {
					t.Fatalf("amount %.0f%% did not increase effect energy", amount*100)
				}
				previous = wetEnergy
			}
		})
	}
}

func TestChorusModulatesOverTime(t *testing.T) {
	// Widely separated impulses should encounter different LFO phases. A
	// fixed echo would produce identical responses at the same relative time.
	input := make([]float32, testSampleRate+4800)
	input[0], input[testSampleRate] = 0.5, 0.5
	output := processChunks(readyProcessor(Settings{Chorus: 1}), input, 512)
	var difference float64
	for i := 480; i < 1920; i++ {
		delta := float64(output[i] - output[testSampleRate+i])
		difference += delta * delta
	}
	if difference < 1e-4 {
		t.Fatal("chorus delay does not modulate over time")
	}
}

func TestEffectsInPlaceAndChunkIndependence(t *testing.T) {
	input := testSignal(12000)
	settings := Settings{Chorus: 0.65, Reverb: 0.8}
	expected := processChunks(readyProcessor(settings), input, len(input))
	for _, chunkSize := range []int{1, 127, 512, 3000} {
		actual := processChunks(readyProcessor(settings), input, chunkSize)
		for i := range actual {
			if actual[i] != expected[i] {
				t.Fatalf("chunk size %d changed sample %d", chunkSize, i)
			}
		}
	}
	inPlace := append([]float32(nil), input...)
	readyProcessor(settings).Process(inPlace, inPlace)
	for i := range inPlace {
		if inPlace[i] != expected[i] {
			t.Fatalf("in-place processing changed sample %d", i)
		}
	}
}

func TestLiveAmountsSmoothAndReturnToBypass(t *testing.T) {
	p := NewProcessor(testSampleRate)
	p.SetSettings(Settings{Chorus: 1, Reverb: 1})
	input := make([]float32, 100)
	for i := range input {
		input[i] = 0.5
	}
	output := processChunks(p, input, 100)
	if math.Abs(float64(output[0]-input[0])) > 0.002 {
		t.Fatal("effect amount jumped instead of ramping at the start")
	}
	if output[len(output)-1] >= output[0] {
		t.Fatal("amount did not move toward its target")
	}
	processChunks(p, testSignal(24000), 512)
	p.SetSettings(Settings{})
	dry := testSignal(4800)
	output = processChunks(p, dry, 127)
	// The wet tails must be inaudible once the 20 ms fade to zero has settled.
	for i := 1440; i < len(output); i++ {
		if output[i] != dry[i] {
			t.Fatalf("effect remained audible at 0%% after ramp, sample %d", i)
		}
	}
}

func TestEffectsRemainBoundedAtFullAmount(t *testing.T) {
	input := testSignal(2 * testSampleRate)
	for i := 0; i < testSampleRate; i++ {
		input[i] = 1 // Sustained input also exercises feedback gain and damping.
	}
	output := processChunks(readyProcessor(Settings{Chorus: 1, Reverb: 1}), input, 512)
	assertFiniteAndBounded(t, output)
	if energy(output) == 0 {
		t.Fatal("effects produced no output")
	}
}

func TestEffectAmountsAreClamped(t *testing.T) {
	p := NewProcessor(testSampleRate)
	p.SetSettings(Settings{Chorus: -1, Reverb: 2})
	if got := p.Settings(); got.Chorus != 0 || got.Reverb != 1 {
		t.Fatalf("out-of-range settings were not clamped: %+v", got)
	}
	p.SetSettings(Settings{Chorus: math.NaN(), Reverb: math.Inf(1)})
	if got := p.Settings(); got.Chorus != 0 || got.Reverb != 1 {
		t.Fatalf("non-finite settings were not clamped: %+v", got)
	}
}

func TestEffectsConcurrentSettings(t *testing.T) {
	p := NewProcessor(testSampleRate)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 2000; i++ {
			p.SetSettings(Settings{
				Chorus: float64(i%101) / 100,
				Reverb: float64((i*3)%101) / 100,
			})
			_ = p.Settings()
		}
	}()
	output := processChunks(p, testSignal(testSampleRate), 128)
	workers.Wait()
	assertFiniteAndBounded(t, output)
}
