package tune

import (
	"encoding/binary"
	"math"
	"os"
	"testing"

	"github.com/heronhoga/auto-tuner/utils/yin"
)

func generateSine(freq, sampleRate float64, numSamples int) []float32 {
	out := make([]float32, numSamples)
	for i := range out {
		out[i] = float32(0.5 * math.Sin(2*math.Pi*freq*float64(i)/sampleRate))
	}
	return out
}

func generateSilenceSine(numSamples int) []float32 {
	out := make([]float32, numSamples)

	for i := range out {
		out[i] = 0
	}

	return out
}

// writeWAV writes 16-bit PCM mono WAV — good enough to open in Audacity.
func writeWAV(path string, samples []float32, sampleRate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	numSamples := len(samples)
	dataSize := numSamples * 2 // 16-bit = 2 bytes/sample
	byteRate := sampleRate * 2

	writeStr := func(s string) { f.WriteString(s) }
	writeU32 := func(v uint32) { binary.Write(f, binary.LittleEndian, v) }
	writeU16 := func(v uint16) { binary.Write(f, binary.LittleEndian, v) }

	writeStr("RIFF")
	writeU32(uint32(36 + dataSize))
	writeStr("WAVE")
	writeStr("fmt ")
	writeU32(16)
	writeU16(1) // PCM
	writeU16(1) // mono
	writeU32(uint32(sampleRate))
	writeU32(uint32(byteRate))
	writeU16(2)  // block align
	writeU16(16) // bits per sample
	writeStr("data")
	writeU32(uint32(dataSize))

	for _, s := range samples {
		clamped := s
		if clamped > 1 {
			clamped = 1
		}
		if clamped < -1 {
			clamped = -1
		}
		binary.Write(f, binary.LittleEndian, int16(clamped*32767))
	}
	return nil
}

// runThroughProcessor simulates a real audio callback feeding small chunks
// (smaller than the hop size) into Process(), and collects all output.
func runThroughProcessor(t *testing.T, input []float32, sampleRate float64, ratio float64, chunkSize int) []float32 {
	t.Helper()

	p := NewPSOLAProcessor(sampleRate, ratio)
	var output []float32

	for start := 0; start < len(input); start += chunkSize {
		end := start + chunkSize
		if end > len(input) {
			end = len(input)
		}
		chunk := input[start:end]

		inPtrs := make([]*float32, len(chunk))
		for i := range chunk {
			inPtrs[i] = &chunk[i]
		}

		outVals := make([]float32, len(chunk))
		outPtrs := make([]*float32, len(chunk))
		for i := range outPtrs {
			outPtrs[i] = &outVals[i]
		}

		p.Process(inPtrs, outPtrs)
		output = append(output, outVals...)

		for _, v := range outVals {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatalf("NaN/Inf in output at global sample ~%d", start)
			}
		}
	}

	return output
}

func TestOLARatioOne(t *testing.T) {
	const sampleRate = 48000.0
	const freq = 440.0
	const durationSec = 1.0
	const chunkSize = 512

	input := generateSine(freq, sampleRate, int(sampleRate*durationSec))
	output := runThroughProcessor(t, input, sampleRate, 1.0, chunkSize)

	if err := writeWAV("test_output_ratio1.wav", output, int(sampleRate)); err != nil {
		t.Fatalf("failed to write wav: %v", err)
	}

	warmup := psolaWindowSize
	cooldown := psolaWindowSize
	if len(output) <= warmup+cooldown {
		t.Fatalf("output too short: %d samples", len(output))
	}
	steadyState := output[warmup : len(output)-cooldown]

	var sumSq float64
	for _, s := range steadyState {
		sumSq += float64(s) * float64(s)
	}
	rms := math.Sqrt(sumSq / float64(len(steadyState)))
	t.Logf("steady-state RMS: %.4f (input RMS ≈ 0.354)", rms)
	if rms < 0.1 || rms > 1.0 {
		t.Errorf("RMS %.4f is way off expected ~0.354 — normalization is likely broken", rms)
	}

	maxExpectedDelta := 2 * math.Pi * freq / sampleRate * 0.5 * 1.5
	var maxDelta float64
	var maxDeltaIndex int
	var clickCount int
	var clickIndices []int
	for i := 1; i < len(steadyState); i++ {
		d := math.Abs(float64(steadyState[i] - steadyState[i-1]))
		if d > maxDelta {
			maxDelta = d
			maxDeltaIndex = i
		}
		if d > maxExpectedDelta*5 {
			clickCount++
			clickIndices = append(clickIndices, i)
		}
	}
	t.Logf("max sample-to-sample delta: %.5f at steadyState index %d (global sample ~%d, hop #%d)",
		maxDelta, maxDeltaIndex, maxDeltaIndex+warmup, (maxDeltaIndex+warmup)/psolaHopSize)
	t.Logf("click-like discontinuities: %d", clickCount)
	for _, idx := range clickIndices {
		t.Logf("  click at steadyState index %d (global sample ~%d, hop #%d): %.5f -> %.5f",
			idx, idx+warmup, (idx+warmup)/psolaHopSize, steadyState[idx-1], steadyState[idx])
	}
	if clickCount > 0 {
		t.Errorf("found %d likely click artifacts — check hop-boundary OLA/normalization math", clickCount)
	}

	hop := psolaHopSize
	var minRMS, maxRMS float64 = math.MaxFloat64, 0
	var minRMSHopIndex int
	hopIdx := 0
	for start := 0; start+hop <= len(steadyState); start += hop {
		seg := steadyState[start : start+hop]
		var s float64
		for _, v := range seg {
			s += float64(v) * float64(v)
		}
		segRMS := math.Sqrt(s / float64(len(seg)))
		t.Logf("hop #%d @ steadyState[%d:%d] (global ~%d): RMS %.4f",
			hopIdx, start, start+hop, start+warmup, segRMS)
		if segRMS < minRMS {
			minRMS = segRMS
			minRMSHopIndex = hopIdx
		}
		if segRMS > maxRMS {
			maxRMS = segRMS
		}
		hopIdx++
	}
	t.Logf("per-hop RMS range: %.4f – %.4f (min at hop #%d)", minRMS, maxRMS, minRMSHopIndex)
	if maxRMS > 0 && (maxRMS-minRMS)/maxRMS > 0.3 {
		t.Errorf("RMS varies %.0f%% across hops — volume ripple suggests normalization bug", (maxRMS-minRMS)/maxRMS*100)
	}
}

func TestOLARatioOneSilence(t *testing.T) {
	const sampleRate = 48000.0
	const freq = 440.0
	const durationSec = 1.0
	const chunkSize = 512

	input := generateSilenceSine(int(sampleRate*durationSec))
	output := runThroughProcessor(t, input, sampleRate, 1.0, chunkSize)

	if err := writeWAV("test_output_ratio1.wav", output, int(sampleRate)); err != nil {
		t.Fatalf("failed to write wav: %v", err)
	}

	warmup := psolaWindowSize
	cooldown := psolaWindowSize
	if len(output) <= warmup+cooldown {
		t.Fatalf("output too short: %d samples", len(output))
	}
	steadyState := output[warmup : len(output)-cooldown]

	// --- 1. RMS sanity check ---
	var sumSq float64
	for _, s := range steadyState {
		sumSq += float64(s) * float64(s)
	}
	rms := math.Sqrt(sumSq / float64(len(steadyState)))
	t.Logf("steady-state RMS: %.4f (input RMS ≈ 0`)", rms)
	if rms > 0.01 {
		t.Errorf("expected near-silence, got RMS %.4f — check for leakage/bug in passthrough path", rms)
	}

	maxExpectedDelta := 2 * math.Pi * freq / sampleRate * 0.5 * 1.5
	var maxDelta float64
	var maxDeltaIndex int
	var clickCount int
	var clickIndices []int
	for i := 1; i < len(steadyState); i++ {
		d := math.Abs(float64(steadyState[i] - steadyState[i-1]))
		if d > maxDelta {
			maxDelta = d
			maxDeltaIndex = i
		}
		if d > maxExpectedDelta*5 {
			clickCount++
			clickIndices = append(clickIndices, i)
		}
	}
	t.Logf("max sample-to-sample delta: %.5f at steadyState index %d (global sample ~%d, hop #%d)",
		maxDelta, maxDeltaIndex, maxDeltaIndex+warmup, (maxDeltaIndex+warmup)/psolaHopSize)
	t.Logf("click-like discontinuities: %d", clickCount)
	for _, idx := range clickIndices {
		t.Logf("  click at steadyState index %d (global sample ~%d, hop #%d): %.5f -> %.5f",
			idx, idx+warmup, (idx+warmup)/psolaHopSize, steadyState[idx-1], steadyState[idx])
	}
	if clickCount > 0 {
		t.Errorf("found %d likely click artifacts — check hop-boundary OLA/normalization math", clickCount)
	}

	hop := psolaHopSize
	var minRMS, maxRMS float64 = math.MaxFloat64, 0
	var minRMSHopIndex int
	hopIdx := 0
	for start := 0; start+hop <= len(steadyState); start += hop {
		seg := steadyState[start : start+hop]
		var s float64
		for _, v := range seg {
			s += float64(v) * float64(v)
		}
		segRMS := math.Sqrt(s / float64(len(seg)))
		t.Logf("hop #%d @ steadyState[%d:%d] (global ~%d): RMS %.4f",
			hopIdx, start, start+hop, start+warmup, segRMS)
		if segRMS < minRMS {
			minRMS = segRMS
			minRMSHopIndex = hopIdx
		}
		if segRMS > maxRMS {
			maxRMS = segRMS
		}
		hopIdx++
	}
	t.Logf("per-hop RMS range: %.4f – %.4f (min at hop #%d)", minRMS, maxRMS, minRMSHopIndex)
	if maxRMS > 0 && (maxRMS-minRMS)/maxRMS > 0.3 {
		t.Errorf("RMS varies %.0f%% across hops — volume ripple suggests normalization bug", (maxRMS-minRMS)/maxRMS*100)
	}
}

func TestPSOLAPitchShift(t *testing.T) {
	const sampleRate = 48000.0
	const inputFreq = 440.0    // A4
	const targetFreq = 466.164 // A#4/Bb4
	const durationSec = 1.0
	const chunkSize = 512

	ratio := targetFreq / inputFreq
	t.Logf("shifting %.2f Hz -> %.2f Hz, ratio = %.5f", inputFreq, targetFreq, ratio)

	input := generateSine(inputFreq, sampleRate, int(sampleRate*durationSec))
	output := runThroughProcessor(t, input, sampleRate, ratio, chunkSize)

	if err := writeWAV("test_output_pitchshift.wav", output, int(sampleRate)); err != nil {
		t.Fatalf("failed to write wav: %v", err)
	}

	warmup := 2*psolaWindowSize + psolaHopSize
	cooldown := psolaWindowSize
	if len(output) <= warmup+cooldown {
		t.Fatalf("output too short: %d samples", len(output))
	}
	steadyState := output[warmup : len(output)-cooldown]

	var sumSq float64
	for _, s := range steadyState {
		sumSq += float64(s) * float64(s)
	}
	rms := math.Sqrt(sumSq / float64(len(steadyState)))
	t.Logf("steady-state RMS: %.4f", rms)
	if rms < 0.1 || rms > 1.0 {
		t.Fatalf("RMS %.4f out of sane range — skipping pitch check, fix amplitude first", rms)
	}

	det := yin.New(sampleRate)
	var detected []float64
	step := psolaWindowSize
	for start := 0; start+psolaWindowSize <= len(steadyState); start += step {
		window := steadyState[start : start+psolaWindowSize]
		note, err := det.Detect(window)
		if err != nil {
			t.Logf("window @ steadyState[%d:%d]: no pitch detected (%v)", start, start+psolaWindowSize, err)
			continue
		}
		t.Logf("window @ steadyState[%d:%d]: detected %.2f Hz", start, start+psolaWindowSize, note.Frequency)
		detected = append(detected, note.Frequency)
	}

	if len(detected) == 0 {
		t.Fatalf("no pitch detected in any steady-state window — cannot verify shift")
	}

	var sum float64
	for _, f := range detected {
		sum += f
	}
	avgDetected := sum / float64(len(detected))
	t.Logf("average detected frequency: %.2f Hz (target %.2f Hz, original %.2f Hz)", avgDetected, targetFreq, inputFreq)

	const tolerance = 5.0 // Hz
	if math.Abs(avgDetected-targetFreq) > tolerance {
		t.Errorf("detected pitch %.2f Hz is not within %.1f Hz of target %.2f Hz",
			avgDetected, tolerance, targetFreq)
	}
	if math.Abs(avgDetected-inputFreq) < tolerance {
		t.Errorf("detected pitch %.2f Hz is still too close to original %.2f Hz — pitch likely did not shift",
			avgDetected, inputFreq)
	}
}