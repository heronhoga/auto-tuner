package ui

import (
	"image/color"

	"fyne.io/fyne/v2/canvas"
)

type WaveformWidget struct {
	Waveform *Waveform
	lines []*canvas.Line
}

func NewWaveformWidget(w *Waveform) *WaveformWidget {
	lines := make([]*canvas.Line, 799)

	for i := range lines {
		line := canvas.NewLine(color.White)
		line.StrokeWidth = 1
		lines[i] = line
	}

	return &WaveformWidget{
		Waveform: w,
		lines: lines,
	}
}