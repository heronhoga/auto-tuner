package oscilloscope

import (
	"sync"

	"fyne.io/fyne/v2/widget"
	utils "github.com/heronhoga/auto-tuner/utils/audio"
	"github.com/heronhoga/auto-tuner/utils/yin"
)

type Widget struct {
	widget.BaseWidget
	RingBuffer *utils.RingBuffer
	mu sync.RWMutex
	NoteData yin.NoteResult
}

func New(ring *utils.RingBuffer) *Widget {
	w := &Widget{
		RingBuffer: ring,
	}

	w.ExtendBaseWidget(w)

	return w
}

func (w *Widget) SetNoteResult (res yin.NoteResult) {
	w.mu.Lock()
	w.NoteData = res
	w.mu.Unlock()
	w.Refresh()
}

func (w *Widget) GetNoteResult() yin.NoteResult {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.NoteData
}
