package oscilloscope

import (
	"fyne.io/fyne/v2/widget"
	utils "github.com/heronhoga/auto-tuner/utils/audio"
)

type Widget struct {
	widget.BaseWidget
	
	RingBuffer *utils.RingBuffer
}

func New(ring *utils.RingBuffer) *Widget {
	w := &Widget{
		RingBuffer: ring,
	}

	w.ExtendBaseWidget(w)

	return w
}
