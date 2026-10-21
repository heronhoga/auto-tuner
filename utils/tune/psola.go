package tune

import (
	"errors"
	"math"
	"sort"

	"github.com/heronhoga/auto-tuner/utils/yin"
)

type PSOLAProcessor struct {
	SampleRate float64
	Ratio      float64

	inputBuffer       []float32
	olaBuffer         []float32
	olaWeights        []float32
	windowWeights     []float32
	windowStart       int
	sampleCursor      int
	nextSynthesisMark float64
	synthesizing      bool
	detector          *yin.Detector
	ratioForFrequency func(float64) float64
}

// Process retains the pointer-based API for existing callers.
func (p *PSOLAProcessor) Process(input []*float32, output []*float32) {
	for i, sample := range input {
		var value float32
		if sample != nil {
			value = *sample
		}
		result := p.processSample(value)
		if i < len(output) && output[i] != nil {
			*output[i] = result
		}
	}
}

// ProcessSamples requires output to hold at least len(input) samples. Analysis
// runs at fixed sample boundaries, independent of the audio callback size.
func (p *PSOLAProcessor) ProcessSamples(input, output []float32) {
	for i, sample := range input {
		output[i] = p.processSample(sample)
	}
}

func (p *PSOLAProcessor) processSample(sample float32) float32 {
	p.inputBuffer = append(p.inputBuffer, sample)
	if len(p.inputBuffer) == psolaWindowSize {
		p.processWindow(p.inputBuffer)
		copy(p.inputBuffer, p.inputBuffer[psolaHopSize:])
		p.inputBuffer = p.inputBuffer[:psolaWindowSize-psolaHopSize]
		p.windowStart += psolaHopSize
	}
	position := p.sampleCursor - psolaWindowSize
	p.sampleCursor++
	if position < 0 {
		return 0
	}
	index := position % len(p.olaBuffer)
	var result float32
	if weight := p.olaWeights[index]; weight > 0 {
		result = p.olaBuffer[index] / weight
	}
	// Recycle the ring slot; memory usage stays bounded during live playback.
	p.olaBuffer[index], p.olaWeights[index] = 0, 0
	return result
}

const (
	psolaWindowSize = 4096
	psolaHopSize    = 2048
)

func NewPSOLAProcessor(sampleRate float64, ratio float64) *PSOLAProcessor {
	p := &PSOLAProcessor{
		SampleRate:    sampleRate,
		Ratio:         1,
		inputBuffer:   make([]float32, 0, psolaWindowSize),
		olaBuffer:     make([]float32, 2*psolaWindowSize),
		olaWeights:    make([]float32, 2*psolaWindowSize),
		windowWeights: make([]float32, psolaWindowSize),
		detector:      yin.New(sampleRate),
	}
	for i := range p.windowWeights {
		// Half-sample offset avoids zero weights at the beginning of the stream.
		p.windowWeights[i] = float32(0.5 - 0.5*math.Cos(2*math.Pi*(float64(i)+0.5)/psolaWindowSize))
	}
	p.SetRatio(ratio)
	return p
}

func (p *PSOLAProcessor) SetRatio(ratio float64) {
	if ratio <= 0 ||
		math.IsNaN(ratio) ||
		math.IsInf(ratio, 0) {
		return
	}

	p.Ratio = ratio
}

func (p *PSOLAProcessor) processWindow(samples []float32) {
	note, err := p.detector.Detect(samples)
	if err != nil {
		p.passthrough(samples)
		return
	}
	ratio := p.Ratio
	if p.ratioForFrequency != nil {
		ratio = p.ratioForFrequency(note.Frequency)
	}
	period := p.SampleRate / note.Frequency
	// A grain must fit within the analysis look-ahead. Unvoiced/out-of-range
	// material and unity ratio use normalized, time-aligned dry overlap-add.
	if period < 2 || period > psolaWindowSize/4 || math.Abs(ratio-1) < 1e-6 {
		p.passthrough(samples)
		return
	}
	marks := FindPitchMark(samples, period)
	if len(marks) < 2 {
		p.passthrough(samples)
		return
	}

	start := p.windowStart + psolaWindowSize/4
	end := start + psolaHopSize
	if !p.synthesizing {
		localStart := start - p.windowStart
		if p.windowStart == 0 {
			localStart = 0
		}
		first := sort.SearchInts(marks, localStart)
		if first == len(marks) {
			p.passthrough(samples)
			return
		}
		p.nextSynthesisMark = float64(p.windowStart + marks[first])
		p.synthesizing = true
	}
	// Keep synthesis phase across windows and repeat/skip source grains to
	// change pitch without changing duration or leaving holes between blocks.
	for p.nextSynthesisMark < float64(end) {
		mark := nearestMark(marks, int(math.Round(p.nextSynthesisMark))-p.windowStart)
		frame := extractAnalysisFrame(samples, mark, period)
		p.accumulate(frame.Samples, hannWindow(len(frame.Samples)), int(math.Round(p.nextSynthesisMark))-frame.Mark)
		p.nextSynthesisMark += period / ratio
	}
}

func (p *PSOLAProcessor) passthrough(samples []float32) {
	p.synthesizing = false
	p.accumulate(samples, p.windowWeights, p.windowStart)
}

func (p *PSOLAProcessor) accumulate(samples, weights []float32, start int) {
	for i, sample := range samples {
		position := start + i
		if position < 0 || position < p.sampleCursor-psolaWindowSize {
			continue
		}
		index := position % len(p.olaBuffer)
		p.olaBuffer[index] += sample * weights[i]
		p.olaWeights[index] += weights[i]
	}
}

func nearestMark(marks []int, position int) int {
	i := sort.SearchInts(marks, position)
	if i == len(marks) || (i > 0 && position-marks[i-1] <= marks[i]-position) {
		return marks[i-1]
	}
	return marks[i]
}

func PSOLA(
	samples []float32,
	analysisMarks []int,
	analysisPeriod float64,
	ratio float64,
) ([]AnalysisFrame, []int, error) {
	if len(samples) == 0 || len(analysisMarks) == 0 || ratio <= 0 {
		return nil, nil, errors.New("invalid PSOLA input")
	}

	frames := extractAnalysisFrames(samples, analysisMarks, analysisPeriod)
	if len(frames) == 0 {
		return nil, nil, errors.New("no analysis frames extracted")
	}

	synthesisPeriod := analysisPeriod / ratio
	synthesisMarks := generateSynthesisMarks(analysisMarks[0], len(frames), synthesisPeriod)

	return frames, synthesisMarks, nil
}
