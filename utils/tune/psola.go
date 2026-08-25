package tune

import (
	"errors"
	"math"

	"github.com/heronhoga/auto-tuner/utils/yin"
)

type PSOLAProcessor struct {
	SampleRate float64
	Ratio      float64

	inputBuffer []float32

	olaBuffer   []float32
	olaWeights  []float32 
	writeCursor int       
	readCursor  int       

	detector *yin.Detector
}

func (p *PSOLAProcessor) Process(input []*float32, output []*float32) {
	for _, sample := range input {
		if sample == nil {
			p.inputBuffer = append(p.inputBuffer, 0)
			continue
		}
		p.inputBuffer = append(p.inputBuffer, *sample)
	}

	for len(p.inputBuffer) >= psolaWindowSize {
		window := p.inputBuffer[:psolaWindowSize]

		frames, marks, err := p.processWindow(window)
		if err == nil {
			p.accumulateOLA(frames, marks)
		}

		p.inputBuffer = p.inputBuffer[psolaHopSize:]
		p.writeCursor += psolaHopSize
	}

	p.drainTo(output)
}

func (p *PSOLAProcessor) ensureCapacity(upTo int) {
	if upTo <= len(p.olaBuffer) {
		return
	}
	grownBuf := make([]float32, upTo)
	copy(grownBuf, p.olaBuffer)
	p.olaBuffer = grownBuf

	grownW := make([]float32, upTo)
	copy(grownW, p.olaWeights)
	p.olaWeights = grownW
}

func (p *PSOLAProcessor) accumulateOLA(frames []AnalysisFrame, synthesisMarks []int) {
	for i, frame := range frames {
		if i >= len(synthesisMarks) {
			break
		}
		globalStart := p.writeCursor + synthesisMarks[i] - frame.Mark
		if globalStart < 0 {
			continue
		}

		p.ensureCapacity(globalStart + len(frame.Samples))

		w := hannWindow(len(frame.Samples))
		for j, sample := range frame.Samples {
			idx := globalStart + j
			p.olaBuffer[idx] += sample
			p.olaWeights[idx] += w[j]
		}
	}
}

func (p *PSOLAProcessor) drainTo(output []*float32) {
	safeUpTo := p.writeCursor - psolaWindowSize
	if safeUpTo < 0 {
		safeUpTo = 0
	}

	for _, out := range output {
		if out == nil {
			continue
		}
		if p.readCursor >= safeUpTo || p.readCursor >= len(p.olaBuffer) {
			*out = 0
			continue
		}
		if w := p.olaWeights[p.readCursor]; w > 1e-6 {
			*out = p.olaBuffer[p.readCursor] / w
		} else {
			*out = 0
		}
		p.readCursor++
	}
}

const (
	psolaWindowSize = 4096
	psolaHopSize    = 2048
)

func NewPSOLAProcessor(sampleRate float64, ratio float64) *PSOLAProcessor {
	return &PSOLAProcessor{
		SampleRate:   sampleRate,
		Ratio:        ratio,
		inputBuffer:  make([]float32, 0),
		detector:     yin.New(sampleRate),
	}
}

func (p *PSOLAProcessor) SetRatio(ratio float64) {
	if ratio <= 0 ||
		math.IsNaN(ratio) ||
		math.IsInf(ratio, 0) {
		return
	}

	p.Ratio = ratio
}

func (p *PSOLAProcessor) processWindow(samples []float32) ([]AnalysisFrame, []int, error) {
	if len(samples) < psolaWindowSize {
		return nil, nil, errors.New("not enough samples for PSOLA")
	}

	note, err := p.detector.Detect(samples)
	if err != nil {
		return passthroughFrame(samples), []int{len(samples) / 2}, nil
	}

	frequency := note.Frequency
	if frequency <= 0 || math.IsNaN(frequency) || math.IsInf(frequency, 0) {
		return passthroughFrame(samples), []int{len(samples) / 2}, nil
	}

	period := p.SampleRate / frequency
	if period <= 0 || math.IsNaN(period) || math.IsInf(period, 0) {
		return nil, nil, errors.New("invalid pitch period")
	}

	marks := FindPitchMark(samples, period)
	if len(marks) < 2 {
		return passthroughFrame(samples), []int{len(samples) / 2}, nil
	}

	return PSOLA(samples, marks, period, p.Ratio)
}

func passthroughFrame(samples []float32) []AnalysisFrame {
	cp := make([]float32, len(samples))
	copy(cp, samples)
	applyHannWindow(cp)

	return []AnalysisFrame{{Samples: cp, Mark: len(samples) / 2}}
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