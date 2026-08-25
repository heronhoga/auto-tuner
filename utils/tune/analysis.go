package tune

import "math"

type AnalysisFrame struct {
	Samples []float32
	Mark    int
}

func hannWindow(n int) []float32 {
	w := make([]float32, n)
	if n <= 1 {
		if n == 1 {
			w[0] = 1
		}
		return w
	}
	for i := 0; i < n; i++ {
		w[i] = float32(0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1))))
	}
	return w
}

func applyHannWindow(samples []float32) {
	n := len(samples)

	if n <= 1 {
		return
	}

	for i := 0; i < n; i++ {
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))

		samples[i] *= float32(window)
	}
}

func extractAnalysisFrame(samples []float32, mark int, period float64) AnalysisFrame {
	windowSize := int(math.Round(period))

	half := windowSize / 2

	start := mark - half
	end := start + windowSize

	if start < 0 {
		start = 0
	}

	if end > len(samples) {
		end = len(samples)
	}

	frame := make([]float32, end-start)

	copy(frame, samples[start:end])

	return AnalysisFrame{
		Samples: frame,
		Mark:    mark - start,
	}
}

func extractAnalysisFrames(samples []float32, analysisMarks []int, analysisPeriod float64) []AnalysisFrame {
	frames := make([]AnalysisFrame, 0, len(analysisMarks))

	for _, mark := range analysisMarks {
		frame := extractAnalysisFrame(samples, mark, analysisPeriod)

		applyHannWindow(frame.Samples)

		frames = append(frames, frame)
	}

	return frames
}

func generateSynthesisMarks(start int, count int, period float64) []int {
	marks := make([]int, count)

	for i := 0; i < count; i++ {
		marks[i] = int(math.Round(float64(start) + float64(i) *period))
	}

	return marks
}

func overlapAdd(frames []AnalysisFrame, synthesisMarks []int, outputLength int) []float32 {
	output := make([]float32, outputLength)

	for i, frame := range frames {
		if i >= len(synthesisMarks) {
			break
		}

		synthesisMark := synthesisMarks[i]

		start := synthesisMark - frame.Mark

		for j, sample := range frame.Samples {
			outputIndex := start + j
			if outputIndex < 0 || outputIndex >= len(output) {
				continue
			}

			output[outputIndex] += sample
		}
	}

	return output
}